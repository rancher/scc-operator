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
			return registration, err
		}
		s.sccCredentials.SetRegistrationCredentialsSecretRef(registration)
	}

	return registration, nil
}

func (s *sccOnlineMode) Register(registrationObj *v1.Registration) (suseconnect.RegistrationSystemID, error) {
	// We must always refresh the sccCredentials - this ensures they are current from the secrets
	credentialsErr := s.sccCredentials.Refresh()
	if credentialsErr != nil {
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
	if registrationCode != "" {
		hash := sha256.Sum256([]byte(registrationCode))
		regCodeHash = hex.EncodeToString(hash[:])
	}

	// Restore cached SubscriptionInfo from RegCode secret if missing in status
	s.restoreSubscriptionInfo(registrationObj)

	needRefresh := false
	if registrationCode != "" {
		needRefresh = subscriptionInfoNeedsRefresh(registrationObj.Status.SubscriptionInfo, regCodeHash)
		if needRefresh && registrationObj.Status.SubscriptionInfo != nil && registrationObj.Status.SubscriptionInfo.RegCodeHash != regCodeHash {
			registrationObj.Status.SubscriptionInfo = nil
		}
	} else {
		registrationObj.Status.SubscriptionInfo = nil
	}

	if needRefresh {
		subInfo, err := sccConnection.SubscriptionInfo(registrationCode)
		if err != nil {
			// Degrade gracefully for all subscription info fetch errors to prevent blocking registration
			s.log.Warnf("failed to fetch subscription info: %v. continuing with registration.", err)
		} else {
			mappedInfo, coveredProductNames := mapSubscriptionInfo(subInfo, regCodeHash)
			registrationObj.Status.SubscriptionInfo = mappedInfo
			if len(coveredProductNames) > maxProductClassLength && subInfo != nil {
				s.log.Warnf("product classes list is too large (%d items), truncating list to %d items.", len(subInfo.ProductClasses), maxProductClassLength)
			}

			// Update the RegCode secret with annotations/data or clean them up
			s.updateRegistrationSecret(registrationObj, coveredProductNames)
		}
	}

	// Register this Rancher cluster to SCC
	id, regErr := sccConnection.RegisterOrKeepAlive(registrationCode)
	if regErr != nil {
		regErr = enrichRegistrationError(regErr, registrationObj.Status.SubscriptionInfo)
		// TODO(scc) do we error different based on ID type?
		return id, regErr
	}

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
	s.log.Debugf("received registration ready for activations %q", registrationObj.Name)
	s.log.Debug("registration ", registrationObj)

	credentialsErr := s.sccCredentials.Refresh()
	if credentialsErr != nil {
		return fmt.Errorf("cannot load scc credentials: %w", credentialsErr)
	}

	registrationCode := suseconnect.FetchSccRegistrationCodeFrom(s.secretRepo, registrationObj.Spec.RegistrationRequest.RegistrationCodeSecretRef)
	sccConnection := s.prepareSCCOnlineConnection(s.rancherMetrics, suseconnect.PrepareSccURL(registrationObj))

	metaData, product, err := sccConnection.Activate(registrationCode)
	if err != nil {
		return err
	}
	s.log.Info(metaData)
	s.log.Info(product)

	s.log.Info("Successfully registered activation")

	return nil
}

func (s *sccOnlineMode) PrepareActivatedForKeepalive(registrationObj *v1.Registration) (*v1.Registration, error) {
	v1.RegistrationConditionSccURLReady.True(registrationObj)

	credentialsErr := s.sccCredentials.Refresh()
	if credentialsErr != nil {
		return nil, fmt.Errorf("cannot load scc credentials: %w", credentialsErr)
	}
	sccConnection := s.prepareSCCOnlineConnection(s.rancherMetrics, suseconnect.PrepareSccURL(registrationObj))

	activations, err := sccConnection.ActivationStatus()
	if err != nil {
		return nil, err
	}
	if len(activations) == 0 {
		return nil, fmt.Errorf("no activations found for registration %q", registrationObj.Name)
	}
	// TODO: what if there are more than 1?
	firstActivation := activations[0]

	registrationObj.Status.RegistrationExpiresAt = &metav1.Time{Time: firstActivation.ExpiresAt}
	registrationObj.Status.RegisteredProduct = &firstActivation.Product.FriendlyName
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
	credRefreshErr := s.sccCredentials.Refresh() // We must always refresh the sccCredentials - this ensures they are current from the secrets
	if credRefreshErr != nil {
		return fmt.Errorf("cannot refresh credentials: %w", credRefreshErr)
	}

	regCode := suseconnect.FetchSccRegistrationCodeFrom(s.secretRepo, registrationObj.Spec.RegistrationRequest.RegistrationCodeSecretRef)
	sccConnection := s.prepareSCCOnlineConnection(s.rancherMetrics, suseconnect.PrepareSccURL(registrationObj))

	metaData, product, err := sccConnection.Activate(regCode)
	if err != nil {
		return err
	}
	s.log.Info(metaData)
	s.log.Info(product)

	// If no error, then system is still registered with valid activation status...
	keepAliveErr := sccConnection.KeepAlive()
	if keepAliveErr != nil {
		return keepAliveErr
	}

	s.log.Info("Successfully checked in with SCC")

	return nil
}

func (s *sccOnlineMode) PrepareKeepaliveSucceeded(registration *v1.Registration) (*v1.Registration, error) {
	v1.RegistrationConditionSccURLReady.True(registration)

	// TODO take any post keepalive success steps
	s.log.Debug("preparing keepalive succeeded")
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
		return regCodeErr
	}
	if lifecycle.SecretHasRegCodeFinalizer(regCodeSecret) {
		updateRegCodeSecret := regCodeSecret.DeepCopy()
		updateRegCodeSecret = lifecycle.SecretRemoveRegCodeFinalizer(updateRegCodeSecret)

		_, regCodeErr = s.secretRepo.Controller.Update(updateRegCodeSecret)
		if regCodeErr != nil {
			return regCodeErr
		}
	}

	if err := s.secretRepo.Controller.Delete(regCodeSecretRef.Namespace, regCodeSecretRef.Name, &metav1.DeleteOptions{}); err != nil {
		return err
	}

	return nil
}

func subscriptionInfoNeedsRefresh(subInfo *v1.SubscriptionInfo, regCodeHash string) bool {
	return subInfo == nil || subInfo.RegCodeHash != regCodeHash
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
	if regSecret == nil || regSecret.Annotations == nil {
		return nil, fmt.Errorf("secret or annotations is nil")
	}
	infoStr, ok := regSecret.Annotations[consts.AnnotationSubscriptionInfo]
	if !ok {
		return nil, fmt.Errorf("annotation not found")
	}
	var subInfo *v1.SubscriptionInfo
	if err := json.Unmarshal([]byte(infoStr), &subInfo); err != nil {
		return nil, err
	}
	return subInfo, nil
}

func mapSubscriptionInfo(subInfo *sccreg.SubscriptionInfo, regCodeHash string) (*v1.SubscriptionInfo, []string) {
	if subInfo == nil {
		return nil, nil
	}
	pcs := make([]v1.ProductClass, 0, len(subInfo.ProductClasses))
	coveredProductNames := make([]string, 0, len(subInfo.ProductClasses))
	for i, pc := range subInfo.ProductClasses {
		if i < maxProductClassLength {
			pcs = append(pcs, v1.ProductClass{
				Name:        pc.Name,
				Description: pc.Description,
			})
		}
		if pc.Description != "" {
			coveredProductNames = append(coveredProductNames, pc.Description)
		} else {
			coveredProductNames = append(coveredProductNames, pc.Name)
		}
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
	return res, coveredProductNames
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
	if subInfo, parseErr := getSubscriptionInfoFromSecret(regSecret); parseErr == nil {
		registrationObj.Status.SubscriptionInfo = subInfo
	}
}

func (s *sccOnlineMode) updateRegistrationSecret(registrationObj *v1.Registration, coveredProductNames []string) {
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
		productListStr := strings.Join(coveredProductNames, ", ")
		if regSecretCopy.Data == nil {
			regSecretCopy.Data = make(map[string][]byte)
		}
		if string(regSecretCopy.Data[consts.SecretKeyCoveredProducts]) != productListStr {
			regSecretCopy.Data[consts.SecretKeyCoveredProducts] = []byte(productListStr)
			changed = true
		}
	} else {
		if regSecretCopy.Annotations != nil && regSecretCopy.Annotations[consts.AnnotationSubscriptionInfo] != "" {
			delete(regSecretCopy.Annotations, consts.AnnotationSubscriptionInfo)
			changed = true
		}
		if regSecretCopy.Data != nil && len(regSecretCopy.Data[consts.SecretKeyCoveredProducts]) > 0 {
			delete(regSecretCopy.Data, consts.SecretKeyCoveredProducts)
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
