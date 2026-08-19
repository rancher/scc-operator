package controllers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/SUSE/connect-ng/pkg/connection"
	sccreg "github.com/SUSE/connect-ng/pkg/registration"
	"github.com/rancher/scc-operator/internal/telemetry"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rancher/scc-operator/internal/consts"
	"github.com/rancher/scc-operator/internal/logging"
	"github.com/rancher/scc-operator/internal/repos/secretrepo"
	"github.com/rancher/scc-operator/internal/suseconnect"
	"github.com/rancher/scc-operator/internal/suseconnect/credentials"
	"github.com/rancher/scc-operator/internal/types"
	v1 "github.com/rancher/scc-operator/pkg/apis/scc.cattle.io/v1"
	"github.com/rancher/scc-operator/pkg/controllers/lifecycle"
)

var (
	activiateMu sync.Mutex
)

const (
	maxProductClassLength = 50
	// Product identifiers for Rancher activations
	rancherProductIdentifier      = "rancher"
	rancherPrimeProductIdentifier = "rancher-prime"
)

type sccOnlineMode struct {
	rancherURL     string
	options        *types.RunOptions
	registration   *v1.Registration
	log            logging.StructuredLogger
	sccCredentials *credentials.CredentialSecretsAdapter
	secretRepo     *secretrepo.SecretRepository
	rancherMetrics telemetry.MetricsWrapper
}

func (s *sccOnlineMode) SetRancherMetrics(rancherMetrics telemetry.MetricsWrapper) {
	s.rancherMetrics = rancherMetrics
}

func (s *sccOnlineMode) prepareSCCOnlineConnection(
	rancherMetrics telemetry.MetricsWrapper,
	registrationURL string,
) suseconnect.SccWrapper {
	return suseconnect.OnlineRancherConnection(
		suseconnect.OnlineConnectionParams{
			RancherURL:      s.rancherURL,
			RegistrationURL: registrationURL,
			Options:         suseconnect.DefaultConnectionOptions(s.options.OperatorName, s.options.OperatorMetadata.Version),
		},
		s.sccCredentials.SccCredentials(),
		rancherMetrics,
	)
}

func (s *sccOnlineMode) NeedsRegistration(registrationObj *v1.Registration) bool {
	return lifecycle.RegistrationHasNotStarted(registrationObj) ||
		!registrationObj.HasCondition(v1.RegistrationConditionSccURLReady) ||
		!registrationObj.HasCondition(v1.RegistrationConditionAnnounced)
}

// PrepareForRegister creates the necessary SCC creds secret and secret reference
func (s *sccOnlineMode) PrepareForRegister(registration *v1.Registration) (*v1.Registration, error) {
	if registration.Status.SystemCredentialsSecretRef == nil {
		err := s.sccCredentials.InitSecret()
		if err != nil {
			s.log.Debugf("failed to initialize SCC credentials secret for registration %s: %v", registration.Name, err)
			return registration, err
		}
		s.sccCredentials.SetRegistrationCredentialsSecretRef(registration)
	}

	return registration, nil
}

func (s *sccOnlineMode) Register(registrationObj *v1.Registration) (suseconnect.RegistrationSystemID, error) {
	s.log.Debugf("registering %s with SCC", registrationObj.Name)
	// We must always refresh the sccCredentials - this ensures they are current from the secrets
	credentialsErr := s.sccCredentials.Refresh()
	if credentialsErr != nil {
		s.log.Debugf("failed to refresh SCC credentials for registration %s: %v", registrationObj.Name, credentialsErr)
		return suseconnect.EmptyRegistrationSystemID, credentialsErr
	}

	// Fetch the SCC registration code; for 80% of users this should be a real code
	// The other cases are either:
	//	a. an error and should have had a code, OR
	//	b. BAYG/RMT/etc based Registration and will not use a code
	registrationCode := suseconnect.FetchSccRegistrationCodeFrom(s.secretRepo, registrationObj.Spec.RegistrationRequest.RegistrationCodeSecretRef)

	// Initiate connection to SCC & verify reg code is for Rancher
	sccConnection := s.prepareSCCOnlineConnection(s.rancherMetrics, suseconnect.PrepareSccURL(registrationObj))

	var regCodeHash string
	if registrationCode == "" {
		registrationObj.Status.SubscriptionInfo = nil
		s.updateRegistrationSecret(registrationObj)
	} else {
		hash := sha256.Sum256([]byte(registrationCode))
		regCodeHash = hex.EncodeToString(hash[:])

		// Restore cached SubscriptionInfo from RegCode secret if missing in status
		// in case the registration failed after fetching the subscription info
		s.restoreSubscriptionInfo(registrationObj)

		if subscriptionInfoNeedsRefresh(registrationObj.Status.SubscriptionInfo, regCodeHash) {
			// Only clear if the registration code has actually changed.
			if registrationObj.Status.SubscriptionInfo != nil && registrationObj.Status.SubscriptionInfo.RegCodeHash != regCodeHash {
				registrationObj.Status.SubscriptionInfo = nil
			}

			subInfo, err := sccConnection.SubscriptionInfo(registrationCode)
			if err != nil {
				// warn all subscription info fetch errors to prevent blocking registration
				s.log.Warnf("failed to fetch subscription info: %v. continuing with registration.", err)
			} else {
				mappedInfo := mapSubscriptionInfo(subInfo, regCodeHash)
				registrationObj.Status.SubscriptionInfo = mappedInfo
				if subInfo != nil && len(subInfo.ProductClasses) > maxProductClassLength {
					s.log.Warnf("product classes list is too large (%d items), truncating list to %d items.", len(subInfo.ProductClasses), maxProductClassLength)
				}

				// Update the RegCode secret with annotations/data or clean them up
				s.updateRegistrationSecret(registrationObj)
			}
		}
	}

	// Register this Rancher cluster to SCC
	s.log.Debugf("calling SCC RegisterOrKeepAlive for registration %s", registrationObj.Name)
	id, regErr := sccConnection.RegisterOrKeepAlive(registrationCode)
	if regErr != nil {
		s.log.Debugf("SCC RegisterOrKeepAlive failed for registration %s: %v", registrationObj.Name, regErr)
		regErr = enrichRegistrationError(regErr, registrationObj.Status.SubscriptionInfo)
		// TODO(scc) do we error different based on ID type?
		return id, regErr
	}

	s.log.Debugf("registration %s successfully registered with SCC (system ID: %d)", registrationObj.Name, id)
	return id, nil
}

func (s *sccOnlineMode) PrepareRegisteredForActivation(registration *v1.Registration) (*v1.Registration, error) {
	if registration.Status.SCCSystemID == nil {
		return registration, errors.New("SCC system ID cannot be empty when preparing registered system")
	}
	baseSccURL := consts.BaseURLForSCC()
	if baseSccURL != "" {
		sccSystemURL := fmt.Sprintf("%s/systems/%d", baseSccURL, *registration.Status.SCCSystemID)
		s.log.Debugf("system announced, check %s", sccSystemURL)

		registration.Status.ActivationStatus.SystemURL = &sccSystemURL
		v1.RegistrationConditionSccURLReady.SetStatusBool(registration, false) // This must be false until successful activation too.
		v1.RegistrationConditionSccURLReady.SetMessageIfBlank(registration, fmt.Sprintf("system announced, check %s", sccSystemURL))
	}

	v1.RegistrationConditionAnnounced.SetStatusBool(registration, true)
	v1.ResourceConditionFailure.SetStatusBool(registration, false)
	v1.ResourceConditionReady.SetStatusBool(registration, true)

	return registration, nil
}

func isNonRecoverableHTTPError(err error) bool {
	var sccAPIError *connection.ApiError

	if errors.As(err, &sccAPIError) {
		httpCode := sccAPIError.Code

		// Client errors (except 429 Too Many Requests) are non-recoverable; a few server errors are also non-recoverable
		if (httpCode >= 400 && httpCode < 500 && httpCode != http.StatusTooManyRequests) ||
			httpCode == http.StatusNotImplemented ||
			httpCode == http.StatusHTTPVersionNotSupported ||
			httpCode == http.StatusNotExtended {
			return true
		}
	}
	return false
}

func getHTTPErrorCode(err error) *int {
	var sccAPIError *connection.ApiError

	if errors.As(err, &sccAPIError) {
		httpCode := sccAPIError.Code
		return &httpCode
	}
	return nil
}

type registrationReconcilerApplier func(regApplierIn *v1.Registration, httpCode *int) *v1.Registration

// reconcileNonRecoverableHTTPError can help reconcile the registration state for any API/HTTP error related reasons
func (s *sccOnlineMode) reconcileNonRecoverableHTTPError(registrationIn *v1.Registration, registerErr error, additionalApplier registrationReconcilerApplier) *v1.Registration {
	httpCode := *getHTTPErrorCode(registerErr)
	nowTime := metav1.Now()
	registrationIn.Status.RegistrationProcessedTS = &nowTime
	registrationIn.Status.ActivationStatus.LastValidatedTS = &nowTime

	wrappedErr := fmt.Errorf("non-recoverable HTTP error encountered; to reregister Rancher, resolve connection issues then try again. Original error: %w", registerErr)
	registrationIn = lifecycle.PrepareFailed(registrationIn, wrappedErr)

	if additionalApplier != nil {
		return additionalApplier(registrationIn, &httpCode)
	}

	return registrationIn
}

func (s *sccOnlineMode) ReconcileRegisterError(registrationObj *v1.Registration, registerErr error, phase types.RegistrationPhase) *v1.Registration {
	// Attempt to restore SubscriptionInfo from RegCode secret
	s.restoreSubscriptionInfo(registrationObj)

	registrationObj = lifecycle.PrepareFailed(registrationObj, registerErr)

	if isNonRecoverableHTTPError(registerErr) {
		return s.reconcileNonRecoverableHTTPError(
			registrationObj,
			registerErr,
			func(regApplierIn *v1.Registration, httpCode *int) *v1.Registration {
				preparedErrorReasonCondition := fmt.Sprintf("Error: SCC api call returned %s (%d) status", http.StatusText(*httpCode), httpCode)
				v1.RegistrationConditionAnnounced.SetError(regApplierIn, preparedErrorReasonCondition, registerErr)
				v1.RegistrationConditionSccURLReady.False(regApplierIn)
				v1.RegistrationConditionActivated.False(regApplierIn)
				regApplierIn.SetCurrentCondition(v1.RegistrationConditionAnnounced)

				// Cannot recover from this error so must set failure
				regApplierIn.Status.ActivationStatus.Activated = false

				return regApplierIn
			},
		)
	}

	v1.RegistrationConditionActivated.False(registrationObj)
	if phase <= types.RegistrationForActivation {
		v1.RegistrationConditionAnnounced.False(registrationObj)
		v1.RegistrationConditionSccURLReady.False(registrationObj)
	}

	if phase == types.RegistrationPrepare {
		v1.ResourceConditionFailure.SetError(registrationObj, "failed during secret initialization", registerErr)
	}

	return registrationObj
}

func (s *sccOnlineMode) NeedsActivation(registrationObj *v1.Registration) bool {
	return lifecycle.RegistrationNeedsActivation(registrationObj)
}

func (s *sccOnlineMode) NeedsPreprocessRegistration(_ *v1.Registration) bool {
	// TODO: online implementation of NeedsPreprocessRegistration
	return false
}

func (s *sccOnlineMode) PreprocessRegistration(registrationObj *v1.Registration) (*v1.Registration, error) {
	// TODO: online implementation of PreprocessRegistration
	return registrationObj, nil
}

func (s *sccOnlineMode) ResetToReadyForActivation(registrationObj *v1.Registration) (*v1.Registration, error) {
	registrationObj.Status.ActivationStatus.Activated = false
	registrationObj.Status.ActivationStatus.LastValidatedTS = &metav1.Time{}
	v1.ResourceConditionProgressing.True(registrationObj)
	v1.ResourceConditionReady.False(registrationObj)
	v1.ResourceConditionDone.False(registrationObj)
	v1.RegistrationConditionActivated.False(registrationObj)
	// Set ResourceConditionProgressing as the CurrentCondition since we're resetting the registration process
	registrationObj.SetCurrentCondition(v1.ResourceConditionProgressing)

	return registrationObj, nil
}

func (s *sccOnlineMode) ReadyForActivation(registrationObj *v1.Registration) bool {
	return v1.RegistrationConditionAnnounced.IsTrue(registrationObj)
}

func (s *sccOnlineMode) Activate(registrationObj *v1.Registration) error {
	s.log.Debugf("activating registration %s", registrationObj.Name)

	credentialsErr := s.sccCredentials.Refresh()
	if credentialsErr != nil {
		s.log.Debugf("failed to refresh SCC credentials for registration %s: %v", registrationObj.Name, credentialsErr)
		return fmt.Errorf("cannot load scc credentials: %w", credentialsErr)
	}

	registrationCode := suseconnect.FetchSccRegistrationCodeFrom(s.secretRepo, registrationObj.Spec.RegistrationRequest.RegistrationCodeSecretRef)
	sccConnection := s.prepareSCCOnlineConnection(s.rancherMetrics, suseconnect.PrepareSccURL(registrationObj))

	s.log.Debugf("calling SCC Activate for registration %s", registrationObj.Name)
	metaData, product, err := sccConnection.Activate(registrationCode)
	if err != nil {
		s.log.Debugf("SCC Activate failed for registration %s: %v", registrationObj.Name, err)
		return err
	}
	s.log.Debugf("activation metadata for %s: %v", registrationObj.Name, metaData)
	s.log.Debugf("activation product for %s: %v", registrationObj.Name, product)

	s.log.Infof("Successfully activated registration %s", registrationObj.Name)

	return nil
}

// selectBestActivation selects the most appropriate activation from a list based on:
// 1. Product identifier match (prefer Rancher product)
// 2. Version match with current Rancher version (if available)
// 3. Most recent StartsAt timestamp (newest activation)
func (s *sccOnlineMode) selectBestActivation(activations []*sccreg.Activation, registrationName string) *sccreg.Activation {
	if len(activations) == 0 {
		return nil
	}
	if len(activations) == 1 {
		return activations[0]
	}

	s.log.Debugf("registration %s has %d activations, selecting best match", registrationName, len(activations))

	// Get current Rancher version if available
	var currentVersion string
	if s.rancherMetrics.Data != nil {
		_, version, _ := s.rancherMetrics.GetProductIdentifier()
		currentVersion = version
		s.log.Debugf("current Rancher version: %s", currentVersion)
	}

	// Filter for Rancher product activations only (both "rancher" and "rancher-prime")
	var rancherActivations []*sccreg.Activation
	for _, activation := range activations {
		if activation.Product != nil {
			s.log.Debugf("activation: identifier=%s, version=%s, friendlyName=%s",
				activation.Product.Identifier, activation.Product.Version, activation.Product.FriendlyName)
			if activation.Product.Identifier == rancherProductIdentifier ||
				activation.Product.Identifier == rancherPrimeProductIdentifier {
				rancherActivations = append(rancherActivations, activation)
			}
		}
	}

	// If we filtered down to Rancher-only activations, use that list
	candidateActivations := activations
	if len(rancherActivations) > 0 {
		s.log.Debugf("found %d Rancher product activations out of %d total", len(rancherActivations), len(activations))
		candidateActivations = rancherActivations
	}

	// Try to find exact version match if we have current version
	if currentVersion != "" {
		for _, activation := range candidateActivations {
			if activation.Product != nil && activation.Product.Version == currentVersion {
				s.log.Debugf("selected activation with exact version match: %s v%s (started %v, expires %v)",
					activation.Product.FriendlyName, activation.Product.Version,
					activation.StartsAt, activation.ExpiresAt)
				return activation
			}
		}
		s.log.Debugf("no exact version match for %s, falling back to newest activation", currentVersion)
	}

	// Fall back to newest activation by StartsAt
	newestActivation := candidateActivations[0]
	for _, activation := range candidateActivations[1:] {
		if activation.StartsAt.After(newestActivation.StartsAt) {
			s.log.Debugf("found newer activation: %s v%s (started %v) vs %s v%s (started %v)",
				activation.Product.FriendlyName, activation.Product.Version, activation.StartsAt,
				newestActivation.Product.FriendlyName, newestActivation.Product.Version, newestActivation.StartsAt)
			newestActivation = activation
		}
	}

	s.log.Debugf("selected newest activation: %s v%s (started %v, expires %v)",
		newestActivation.Product.FriendlyName, newestActivation.Product.Version,
		newestActivation.StartsAt, newestActivation.ExpiresAt)

	return newestActivation
}

// needsVersionUpgrade determines if the current Rancher version differs from what's saved in the registration status.
// Returns true if an upgrade is needed (current version differs from ProductVersion in status).
// This avoids an SCC API call by using the locally stored ProductVersion field.
func (s *sccOnlineMode) needsVersionUpgrade(registration *v1.Registration, currentVersion string) bool {
	// If we've never set a product version, no upgrade needed (initial activation will handle it)
	if registration.Status.ActivationStatus.ProductVersion == nil {
		s.log.Debugf("no product version saved in status, skipping version check")
		return false
	}

	savedVersion := *registration.Status.ActivationStatus.ProductVersion
	if savedVersion != currentVersion {
		s.log.Debugf("version mismatch detected: saved=%s, current=%s - upgrade needed", savedVersion, currentVersion)
		return true
	}

	s.log.Debugf("version match: saved=%s, current=%s - no upgrade needed", savedVersion, currentVersion)
	return false
}

// refreshProductMetadata fetches current activation status and updates product-related fields.
// This is called both after initial activation and during keepalive to ensure the registration
// status reflects the current product information from SCC.
func (s *sccOnlineMode) refreshProductMetadata(registration *v1.Registration) error {
	s.log.Debugf("refreshing product metadata for registration %s", registration.Name)

	// Refresh credentials to ensure we have current secrets
	credentialsErr := s.sccCredentials.Refresh()
	if credentialsErr != nil {
		s.log.Debugf("failed to refresh SCC credentials for registration %s: %v", registration.Name, credentialsErr)
		return fmt.Errorf("cannot load scc credentials: %w", credentialsErr)
	}

	sccConnection := s.prepareSCCOnlineConnection(s.rancherMetrics, suseconnect.PrepareSccURL(registration))

	// Fetch current activations from SCC
	s.log.Debugf("fetching activation status for registration %s", registration.Name)
	activations, err := sccConnection.ActivationStatus()
	if err != nil {
		s.log.Debugf("failed to fetch activation status for registration %s: %v", registration.Name, err)
		return err
	}

	if len(activations) == 0 {
		s.log.Debugf("no activations found for registration %s", registration.Name)
		return fmt.Errorf("no activations found for registration %q", registration.Name)
	}

	// Select the best activation based on product, version, and timestamp
	selectedActivation := s.selectBestActivation(activations, registration.Name)
	if selectedActivation == nil {
		return fmt.Errorf("failed to select activation for registration %q", registration.Name)
	}

	s.log.Debugf("registration %s: using product %s v%s (expires at %v)",
		registration.Name, selectedActivation.Product.FriendlyName,
		selectedActivation.Product.Version, selectedActivation.ExpiresAt)

	// Update product metadata fields
	registration.Status.RegistrationExpiresAt = &metav1.Time{Time: selectedActivation.ExpiresAt}
	registration.Status.RegisteredProduct = &selectedActivation.Product.FriendlyName
	registration.Status.ActivationStatus.ProductVersion = &selectedActivation.Product.Version

	return nil
}

func (s *sccOnlineMode) PrepareActivatedForKeepalive(registrationObj *v1.Registration) (*v1.Registration, error) {
	s.log.Debugf("preparing keepalive for registration %s", registrationObj.Name)
	v1.RegistrationConditionSccURLReady.True(registrationObj)

	// Refresh product metadata (expiration date and product name) from SCC
	// Use fatal error handling for initial activation preparation
	err := s.refreshProductMetadata(registrationObj)
	if err != nil {
		return nil, err
	}

	return registrationObj, nil
}

// ReconcileActivateError will first verify if an error is recoverable and then reconcile the error as needed
func (s *sccOnlineMode) ReconcileActivateError(registration *v1.Registration, activationErr error, _ types.ActivationPhase) *v1.Registration {
	if isNonRecoverableHTTPError(activationErr) {
		return s.reconcileNonRecoverableHTTPError(
			registration,
			activationErr,
			func(regApplierIn *v1.Registration, httpCode *int) *v1.Registration {
				preparedErrorReasonCondition := fmt.Sprintf("Error: SCC sync returned %s (%d) status", http.StatusText(*httpCode), httpCode)
				v1.RegistrationConditionActivated.SetError(regApplierIn, preparedErrorReasonCondition, activationErr)
				regApplierIn.SetCurrentCondition(v1.RegistrationConditionActivated)

				// Cannot recover from this error so must set failure
				regApplierIn.Status.ActivationStatus.Activated = false

				return regApplierIn
			},
		)
	}

	// TODO other error reconcile when non-http error based

	// Return the unmodified version
	return registration
}

func (s *sccOnlineMode) Keepalive(registrationObj *v1.Registration) error {
	s.log.Debugf("performing keepalive for registration %s", registrationObj.Name)
	credRefreshErr := s.sccCredentials.Refresh() // We must always refresh the sccCredentials - this ensures they are current from the secrets
	if credRefreshErr != nil {
		s.log.Debugf("failed to refresh SCC credentials for registration %s: %v", registrationObj.Name, credRefreshErr)
		return fmt.Errorf("cannot refresh credentials: %w", credRefreshErr)
	}

	sccConnection := s.prepareSCCOnlineConnection(s.rancherMetrics, suseconnect.PrepareSccURL(registrationObj))

	// Check if Rancher version has changed and upgrade activation if needed
	_, currentVersion, _ := s.rancherMetrics.GetProductIdentifier()
	if s.needsVersionUpgrade(registrationObj, currentVersion) {
		s.log.Infof("Rancher version changed from %s to %s for registration %s, upgrading activation",
			*registrationObj.Status.ActivationStatus.ProductVersion, currentVersion, registrationObj.Name)
		metaData, product, upgradeErr := sccConnection.Upgrade()
		if upgradeErr != nil {
			s.log.Warnf("activation upgrade failed for registration %s: %v - will retry on next keepalive", registrationObj.Name, upgradeErr)
			// Continue with keepalive - upgrade will retry next time
		} else {
			s.log.Infof("Successfully upgraded activation to %s v%s for registration %s", product.FriendlyName, product.Version, registrationObj.Name)
			s.log.Debugf("upgrade metadata for %s: %v", registrationObj.Name, metaData)
		}
	}

	// Perform keepalive heartbeat with SCC
	s.log.Debugf("calling SCC KeepAlive for registration %s", registrationObj.Name)
	keepAliveErr := sccConnection.KeepAlive()
	if keepAliveErr != nil {
		s.log.Debugf("SCC KeepAlive failed for registration %s: %v", registrationObj.Name, keepAliveErr)
		return keepAliveErr
	}

	s.log.Infof("Successfully checked in with SCC for registration %s", registrationObj.Name)

	return nil
}

func (s *sccOnlineMode) PrepareKeepaliveSucceeded(registration *v1.Registration) (*v1.Registration, error) {
	v1.RegistrationConditionSccURLReady.True(registration)

	s.log.Debug("preparing keepalive succeeded")

	// Refresh product metadata to ensure status reflects current product information
	// Use non-fatal error handling - keepalive already succeeded, metadata refresh is supplementary
	err := s.refreshProductMetadata(registration)
	if err != nil {
		// This should never happen with fatalErrors=false, but handle defensively
		s.log.Warnf("unexpected error during metadata refresh for registration %s: %v", registration.Name, err)
	}

	return registration, nil
}

func (s *sccOnlineMode) ReconcileKeepaliveError(registration *v1.Registration, keepaliveErr error) *v1.Registration {
	if isNonRecoverableHTTPError(keepaliveErr) {
		return s.reconcileNonRecoverableHTTPError(
			registration,
			keepaliveErr,
			func(regApplierIn *v1.Registration, httpCode *int) *v1.Registration {
				preparedErrorReasonCondition := fmt.Sprintf("Error: SCC sync returned %s (%d) status", http.StatusText(*httpCode), httpCode)
				v1.RegistrationConditionKeepalive.SetError(regApplierIn, preparedErrorReasonCondition, keepaliveErr)
				regApplierIn.SetCurrentCondition(v1.RegistrationConditionKeepalive)

				// Cannot recover from this error so must set failure
				regApplierIn.Status.ActivationStatus.Activated = false

				return regApplierIn
			},
		)
	}

	// TODO other error reconcile when non-http error based

	return registration
}

func (s *sccOnlineMode) Deregister() error {
	_ = s.sccCredentials.Refresh()
	sccConnection := s.prepareSCCOnlineConnection(s.rancherMetrics, suseconnect.PrepareSccURL(s.registration))
	// TODO : this causes deletion to fail if the credentials are invalid. I think we
	// need to do a best effort check to see if it was ever registered before
	// we want to fail to delete if deregister fails, but the system is registered in SCC

	// Finalizers on the credential secret have helped this case, but it's still invalid if users edit the secret manually for some reason.
	if err := sccConnection.Deregister(); err != nil {
		s.log.Warn("Deregister failure will be logged but not prevent cleanup")
		s.log.Errorf("Failed to deregister SCC registration: %v", err)
	}

	// Delete SCC credentials after successful Deregister
	credErr := s.sccCredentials.Remove()
	if credErr != nil {
		return credErr
	}

	regCodeSecretRef := s.registration.Spec.RegistrationRequest.RegistrationCodeSecretRef
	regCodeSecret, regCodeErr := s.secretRepo.Get(regCodeSecretRef.Namespace, regCodeSecretRef.Name)
	if regCodeErr != nil && !apierrors.IsNotFound(regCodeErr) {
		s.log.Debugf("failed to get registration code secret %s/%s during cleanup: %v", regCodeSecretRef.Namespace, regCodeSecretRef.Name, regCodeErr)
		return regCodeErr
	}
	if lifecycle.SecretHasRegCodeFinalizer(regCodeSecret) {
		updateRegCodeSecret := regCodeSecret.DeepCopy()
		updateRegCodeSecret = lifecycle.SecretRemoveRegCodeFinalizer(updateRegCodeSecret)

		_, regCodeErr = s.secretRepo.Controller.Update(updateRegCodeSecret)
		if regCodeErr != nil {
			s.log.Debugf("failed to remove finalizer from registration code secret %s/%s: %v", regCodeSecretRef.Namespace, regCodeSecretRef.Name, regCodeErr)
			return regCodeErr
		}
	}

	if err := s.secretRepo.Controller.Delete(regCodeSecretRef.Namespace, regCodeSecretRef.Name, &metav1.DeleteOptions{}); err != nil {
		s.log.Debugf("failed to delete registration code secret %s/%s: %v", regCodeSecretRef.Namespace, regCodeSecretRef.Name, err)
		return err
	}

	return nil
}

func subscriptionInfoNeedsRefresh(subInfo *v1.SubscriptionInfo, regCodeHash string) bool {
	if regCodeHash == "" {
		return false
	}
	if subInfo == nil || subInfo.RegCodeHash != regCodeHash {
		return true
	}
	if subInfo.ExpiresAt != nil && !subInfo.ExpiresAt.IsZero() && time.Now().After(subInfo.ExpiresAt.Time) {
		return true
	}
	return false
}

func enrichRegistrationError(regErr error, subInfo *v1.SubscriptionInfo) error {
	if regErr == nil || subInfo == nil || len(subInfo.ProductClasses) == 0 {
		return regErr
	}
	if !isNonRecoverableHTTPError(regErr) {
		return regErr
	}
	var covered []string
	for _, pc := range subInfo.ProductClasses {
		if pc.Description != "" {
			covered = append(covered, pc.Description)
		} else {
			covered = append(covered, pc.Name)
		}
	}
	return fmt.Errorf("the reg code provided is for %s (original error: %w)", strings.Join(covered, ", "), regErr)
}

func getSubscriptionInfoFromSecret(regSecret *corev1.Secret) (*v1.SubscriptionInfo, error) {
	if regSecret == nil {
		return nil, errors.New("no secret specified")
	}
	if regSecret.Annotations == nil {
		return nil, errors.New("no annotations found")
	}

	infoStr, ok := regSecret.Annotations[consts.AnnotationSubscriptionInfo]
	if !ok {
		return nil, errors.New("subscription info annotation not found")
	}

	var subInfo *v1.SubscriptionInfo
	if err := json.Unmarshal([]byte(infoStr), &subInfo); err != nil {
		return nil, err
	}
	return subInfo, nil
}

func mapSubscriptionInfo(subInfo *sccreg.SubscriptionInfo, regCodeHash string) *v1.SubscriptionInfo {
	if subInfo == nil {
		return nil
	}
	pcsLimit := min(len(subInfo.ProductClasses), maxProductClassLength)
	pcs := make([]v1.ProductClass, 0, pcsLimit)
	for i, pc := range subInfo.ProductClasses {
		if i >= maxProductClassLength {
			break
		}
		pcs = append(pcs, v1.ProductClass{
			Name:        pc.Name,
			Description: pc.Description,
		})
	}

	var startsAt *metav1.Time
	if !subInfo.StartsAt.IsZero() {
		startsAt = &metav1.Time{Time: subInfo.StartsAt}
	}
	var expiresAt *metav1.Time
	if !subInfo.ExpiresAt.IsZero() {
		expiresAt = &metav1.Time{Time: subInfo.ExpiresAt}
	}

	res := &v1.SubscriptionInfo{
		Kind:           subInfo.Kind,
		Name:           subInfo.Name,
		StartsAt:       startsAt,
		ExpiresAt:      expiresAt,
		Limit:          subInfo.Limit,
		Notifications:  subInfo.Notifications,
		ProductClasses: pcs,
		RegCodeHash:    regCodeHash,
	}
	return res
}

func (s *sccOnlineMode) restoreSubscriptionInfo(registrationObj *v1.Registration) {
	if registrationObj.Status.SubscriptionInfo != nil || registrationObj.Spec.RegistrationRequest == nil || registrationObj.Spec.RegistrationRequest.RegistrationCodeSecretRef == nil {
		return
	}

	secretRef := registrationObj.Spec.RegistrationRequest.RegistrationCodeSecretRef
	regSecret, err := s.secretRepo.Get(secretRef.Namespace, secretRef.Name)
	if err != nil || regSecret == nil {
		return
	}

	if subInfo, err := getSubscriptionInfoFromSecret(regSecret); err == nil {
		registrationObj.Status.SubscriptionInfo = subInfo
	}
}

func (s *sccOnlineMode) updateRegistrationSecret(registrationObj *v1.Registration) {
	if registrationObj.Spec.RegistrationRequest == nil || registrationObj.Spec.RegistrationRequest.RegistrationCodeSecretRef == nil {
		return
	}
	secretRef := registrationObj.Spec.RegistrationRequest.RegistrationCodeSecretRef
	regSecret, getErr := s.secretRepo.Get(secretRef.Namespace, secretRef.Name)
	if getErr != nil || regSecret == nil {
		s.log.Warnf("failed to get RegCode secret for updating: %v", getErr)
		return
	}

	regSecretCopy := regSecret.DeepCopy()
	changed := false

	if registrationObj.Status.SubscriptionInfo != nil {
		if regSecretCopy.Annotations == nil {
			regSecretCopy.Annotations = make(map[string]string)
		}
		if infoBytes, marshalErr := json.Marshal(registrationObj.Status.SubscriptionInfo); marshalErr == nil {
			newAnn := string(infoBytes)
			if regSecretCopy.Annotations[consts.AnnotationSubscriptionInfo] != newAnn {
				regSecretCopy.Annotations[consts.AnnotationSubscriptionInfo] = newAnn
				changed = true
			}
		}
	} else {
		if regSecretCopy.Annotations != nil && regSecretCopy.Annotations[consts.AnnotationSubscriptionInfo] != "" {
			delete(regSecretCopy.Annotations, consts.AnnotationSubscriptionInfo)
			changed = true
		}
	}

	if changed {
		if _, updateSecretErr := s.secretRepo.CreateOrUpdateSecret(regSecretCopy); updateSecretErr != nil {
			s.log.Warnf("failed to update RegCode secret: %v", updateSecretErr)
		}
	}
}

var _ SCCHandler = &sccOnlineMode{}
