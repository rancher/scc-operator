package controllers

import (
	"fmt"

	"github.com/SUSE/connect-ng/pkg/registration"
	"github.com/rancher/scc-operator/internal/suseconnect/offlinevalidator"
	"github.com/rancher/scc-operator/internal/telemetry"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rootLog "github.com/rancher/scc-operator/internal/log"
	"github.com/rancher/scc-operator/internal/suseconnect"
	offlineSecrets "github.com/rancher/scc-operator/internal/suseconnect/offline"
	"github.com/rancher/scc-operator/internal/types"
	v1 "github.com/rancher/scc-operator/pkg/apis/scc.cattle.io/v1"
	"github.com/rancher/scc-operator/pkg/controllers/shared"
)

type sccOfflineMode struct {
	rancherURL     string
	rancherUUID    string
	options        *types.RunOptions
	registration   *v1.Registration
	log            rootLog.StructuredLogger
	offlineSecrets *offlineSecrets.SecretManager
	productMetrics telemetry.MetricsWrapper
}

const InitialOfflineCertificateReadyMessage = "Awaiting registration certificate secret"

func (s *sccOfflineMode) SetProductMetrics(productMetrics telemetry.MetricsWrapper) {
	s.productMetrics = productMetrics
}

func (s *sccOfflineMode) NeedsRegistration(registrationObj *v1.Registration) bool {
	return registrationObj.Spec.OfflineRegistrationCertificateSecretRef == nil &&
		(shared.RegistrationHasNotStarted(registrationObj) ||
			!registrationObj.HasCondition(v1.RegistrationConditionOfflineRequestReady) ||
			v1.RegistrationConditionOfflineRequestReady.IsFalse(registrationObj))
}

func (s *sccOfflineMode) PrepareForRegister(registrationObj *v1.Registration) (*v1.Registration, error) {
	if registrationObj.Status.OfflineRegistrationRequest == nil {
		err := s.offlineSecrets.InitRequestSecret()
		if err != nil {
			return registrationObj, err
		}
		s.offlineSecrets.SetRegistrationOfflineRegistrationRequestSecretRef(registrationObj)
	}

	return registrationObj, nil
}

func (s *sccOfflineMode) RefreshOfflineRequestSecret() error {
	// TODO: sort out something other than nil
	sccWrapper := suseconnect.OfflineRancherRegistration(s.rancherURL, s.productMetrics)
	generatedOfflineRegistrationRequest, err := sccWrapper.PrepareOfflineRegistrationRequest()
	if err != nil {
		return err
	}
	return s.offlineSecrets.UpdateOfflineRequest(generatedOfflineRegistrationRequest)
}

func (s *sccOfflineMode) Register(_ *v1.Registration) (suseconnect.RegistrationSystemID, error) {
	refreshErr := s.RefreshOfflineRequestSecret()
	if refreshErr != nil {
		return suseconnect.EmptyRegistrationSystemID, refreshErr
	}

	return suseconnect.OfflineRegistrationSystemID, nil
}

func (s *sccOfflineMode) PrepareRegisteredForActivation(registrationObj *v1.Registration) (*v1.Registration, error) {

	v1.RegistrationConditionOfflineRequestReady.True(registrationObj)
	v1.RegistrationConditionOfflineCertificateReady.False(registrationObj)
	v1.RegistrationConditionOfflineCertificateReady.SetMessageIfBlank(registrationObj, InitialOfflineCertificateReadyMessage)
	registrationObj.SetCurrentCondition(v1.RegistrationConditionOfflineCertificateReady)

	return registrationObj, nil
}

// ReconcileRegisterError helps reconcile any errors in the register phase
func (s *sccOfflineMode) ReconcileRegisterError(registrationObj *v1.Registration, registerErr error, phase types.RegistrationPhase) *v1.Registration {
	if phase == types.RegistrationInit {
		v1.RegistrationConditionOfflineRequestReady.SetError(registrationObj, "Failed to prepare Offline Request secret & ref", registerErr)
	}
	if phase == types.RegistrationMain {
		v1.RegistrationConditionOfflineRequestReady.SetError(registrationObj, "Failed to update Offline Request secret", registerErr)
	}
	registrationObj.SetCurrentCondition(v1.RegistrationConditionOfflineRequestReady)
	return registrationObj
}

func (s *sccOfflineMode) NeedsActivation(registrationObj *v1.Registration) bool {
	return registrationObj.Status.OfflineRegistrationRequest != nil &&
		shared.RegistrationNeedsActivation(registrationObj)
}

func (s *sccOfflineMode) ReadyForActivation(registrationObj *v1.Registration) bool {
	return registrationObj.Status.OfflineRegistrationRequest != nil &&
		registrationObj.Spec.OfflineRegistrationCertificateSecretRef != nil
}

func (s *sccOfflineMode) ResetToRegisteredForActivation(registrationObj *v1.Registration) (*v1.Registration, error) {
	registrationObj.RemoveCondition(v1.RegistrationConditionActivated)
	registrationObj.RemoveCondition(v1.RegistrationConditionOfflineCertificateReady)
	registrationObj.RemoveCondition(v1.ResourceConditionFailure)
	registrationObj.RemoveCondition(v1.ResourceConditionReady)

	v1.ResourceConditionProgressing.True(registrationObj)
	registrationObj, prepErr := s.PrepareRegisteredForActivation(registrationObj)
	if prepErr != nil {
		return nil, fmt.Errorf("failed resetting Registration back to RegisteredForActivation, setting conditions failed: %w", prepErr)
	}

	certErr := s.RemoveOfflineCertificate()
	if certErr != nil {
		return nil, fmt.Errorf("failed resetting Registration back to RegisteredForActivation, removing certificate failed: %w", certErr)
	}

	return registrationObj, nil
}

func (s *sccOfflineMode) Activate(_ *v1.Registration) error {
	certReader, err := s.offlineSecrets.OfflineCertificateReader()
	if err != nil {
		return fmt.Errorf("activate failed, cannot get offline certificate reader: %w", err)
	}

	offlineCert, certErr := registration.OfflineCertificateFrom(certReader, false)
	if certErr != nil {
		return fmt.Errorf("activate failed, cannot prepare offline certificate: %w", certErr)
	}

	offlineCertValidator := offlinevalidator.New(offlineCert, s.rancherUUID)

	return offlineCertValidator.ValidateCertificate()
}

func (s *sccOfflineMode) PrepareActivatedForKeepalive(registrationObj *v1.Registration) (*v1.Registration, error) {
	// TODO: can we actually get the SCC systemID in offline mode?
	// GH issue: https://github.com/SUSE/connect-ng/issues/313
	/*
		certReader, err := s.offlineSecrets.OfflineCertificateReader()
		if err != nil {
			return registrationObj, fmt.Errorf("activate failed, cannot get offline certificate reader: %w", err)
		}

		offlineCert, certErr := registration.OfflineCertificateFrom(certReader, false)
		if certErr != nil {
			return registrationObj, fmt.Errorf("activate failed, cannot prepare offline certificate: %w", certErr)
		}
	*/

	registrationObj.RemoveCondition(v1.RegistrationConditionOfflineCertificateReady)
	v1.RegistrationConditionOfflineCertificateReady.True(registrationObj)
	v1.ActivationConditionOfflineDone.True(registrationObj)
	return registrationObj, nil
}

func (s *sccOfflineMode) RemoveOfflineCertificate() error {
	certErr := s.offlineSecrets.RemoveOfflineCertificate()

	if certErr != nil {
		return fmt.Errorf("failed removing offline certificate: %w", certErr)
	}

	return nil
}

func (s *sccOfflineMode) ReconcileActivateError(registrationObj *v1.Registration, activationErr error, _ types.ActivationPhase) *v1.Registration {
	// TODO: this will need updating to use phase after todo inside PrepareActivatedForKeepalive is solved
	v1.RegistrationConditionActivated.False(registrationObj)
	v1.RegistrationConditionActivated.Reason(registrationObj, "offline activation failed")
	v1.RegistrationConditionOfflineCertificateReady.SetError(registrationObj, "cannot validate offline certificate", activationErr)
	registrationObj.SetCurrentCondition(v1.RegistrationConditionOfflineCertificateReady)

	// Cannot recover from this error so must set failure
	registrationObj.Status.ActivationStatus.Activated = false

	return shared.PrepareFailed(registrationObj, activationErr)
}

func (s *sccOfflineMode) Keepalive(registrationObj *v1.Registration) error {
	s.log.Debugf("For now offline keepalive is an intentional noop")
	// TODO: eventually keepalive for offline should mimic `PrepareRegisteredForActivation` creation of ORR (to update metrics for next offline registration)

	expiresAt := registrationObj.Status.RegistrationExpiresAt
	now := metav1.Now()
	if expiresAt.Before(&now) {
		return fmt.Errorf("offline registration has expired; expired at %v before current time (%v)", expiresAt, now)
	}

	certReader, err := s.offlineSecrets.OfflineCertificateReader()
	if err != nil {
		return fmt.Errorf("activate failed, cannot get offline certificate reader: %w", err)
	}

	offlineCert, certErr := registration.OfflineCertificateFrom(certReader, false)
	if certErr != nil {
		return fmt.Errorf("activate failed, cannot prepare offline certificate: %w", certErr)
	}

	offlineCertValidator := offlinevalidator.New(offlineCert, s.rancherUUID)
	validateErr := offlineCertValidator.ValidateCertificate()
	if validateErr != nil {
		return fmt.Errorf("activate failed, cannot validate offline certificate: %w", validateErr)
	}

	return nil
}

func (s *sccOfflineMode) PrepareKeepaliveSucceeded(registrationObj *v1.Registration) (*v1.Registration, error) {
	sccWrapper := suseconnect.OfflineRancherRegistration(s.rancherURL, s.productMetrics)
	generatedOfflineRegistrationRequest, err := sccWrapper.PrepareOfflineRegistrationRequest()
	if err != nil {
		return registrationObj, err
	}
	updateErr := s.offlineSecrets.UpdateOfflineRequest(generatedOfflineRegistrationRequest)
	if updateErr != nil {
		return registrationObj, updateErr
	}

	return registrationObj, nil
}

func (s *sccOfflineMode) ReconcileKeepaliveError(registration *v1.Registration, err error) *v1.Registration {
	s.log.Error(err)
	// TODO: handle errors from Keepalive and PrepareKeepaliveSucceeded
	return registration
}

func (s *sccOfflineMode) Deregister() error {
	delErr := s.offlineSecrets.Remove()
	if delErr != nil {
		return fmt.Errorf("deregister failed: %w", delErr)
	}

	return nil
}

var _ SCCHandler = &sccOfflineMode{}
