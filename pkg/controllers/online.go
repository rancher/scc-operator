package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/SUSE/connect-ng/pkg/connection"
	"github.com/rancher/scc-operator/internal/telemetry"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rancher/scc-operator/internal/consts"
	"github.com/rancher/scc-operator/internal/log"
	"github.com/rancher/scc-operator/internal/repos/secretrepo"
	"github.com/rancher/scc-operator/internal/suseconnect"
	"github.com/rancher/scc-operator/internal/suseconnect/credentials"
	"github.com/rancher/scc-operator/internal/types"
	v1 "github.com/rancher/scc-operator/pkg/apis/scc.cattle.io/v1"
	"github.com/rancher/scc-operator/pkg/controllers/shared"
)

var (
	activiateMu sync.Mutex
)

type sccOnlineMode struct {
	rancherURL     string
	options        *types.RunOptions
	registration   *v1.Registration
	log            log.StructuredLogger
	sccCredentials *credentials.CredentialSecretsAdapter
	secretRepo     *secretrepo.SecretRepository
	productMetrics telemetry.MetricsWrapper
}

func (s *sccOnlineMode) SetProductMetrics(productMetrics telemetry.MetricsWrapper) {
	s.productMetrics = productMetrics
}

func (s *sccOnlineMode) prepareSCCOnlineConnection(
	productMetrics telemetry.MetricsWrapper,
	registrationURL string,
) suseconnect.SccWrapper {
	return suseconnect.OnlineRancherConnection(
		suseconnect.OnlineConnectionParams{
			RancherURL:      s.rancherURL,
			RegistrationURL: registrationURL,
			Options:         suseconnect.DefaultConnectionOptions(s.options.OperatorName, s.options.OperatorMetadata.Version),
		},
		s.sccCredentials.SccCredentials(),
		productMetrics,
	)
}

func (s *sccOnlineMode) NeedsRegistration(registrationObj *v1.Registration) bool {
	return shared.RegistrationHasNotStarted(registrationObj) ||
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
	sccConnection := s.prepareSCCOnlineConnection(s.productMetrics, suseconnect.PrepareSccURL(registrationObj))

	// Register this Rancher cluster to SCC
	id, regErr := sccConnection.RegisterOrKeepAlive(registrationCode)
	if regErr != nil {
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
	registrationIn = shared.PrepareFailed(registrationIn, wrappedErr)

	if additionalApplier != nil {
		return additionalApplier(registrationIn, &httpCode)
	}

	return registrationIn
}

func (s *sccOnlineMode) ReconcileRegisterError(registrationObj *v1.Registration, registerErr error, phase types.RegistrationPhase) *v1.Registration {
	registrationObj = shared.PrepareFailed(registrationObj, registerErr)

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
	return shared.RegistrationNeedsActivation(registrationObj)
}

func (s *sccOnlineMode) ResetToRegisteredForActivation(_ *v1.Registration) (*v1.Registration, error) {
	// TODO: Implement this function equivalent for online mode
	return nil, nil
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
	sccConnection := s.prepareSCCOnlineConnection(s.productMetrics, suseconnect.PrepareSccURL(registrationObj))

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
	sccConnection := s.prepareSCCOnlineConnection(s.productMetrics, suseconnect.PrepareSccURL(registrationObj))

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
	sccConnection := s.prepareSCCOnlineConnection(s.productMetrics, suseconnect.PrepareSccURL(registrationObj))

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
	sccConnection := s.prepareSCCOnlineConnection(s.productMetrics, suseconnect.PrepareSccURL(s.registration))
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
	if shared.SecretHasRegCodeFinalizer(regCodeSecret) {
		updateRegCodeSecret := regCodeSecret.DeepCopy()
		updateRegCodeSecret = shared.SecretRemoveRegCodeFinalizer(updateRegCodeSecret)

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

var _ SCCHandler = &sccOnlineMode{}
