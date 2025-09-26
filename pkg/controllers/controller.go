package controllers

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/rancher/scc-operator/internal/rancher"
	"github.com/rancher/scc-operator/internal/rancher/settings"
	"github.com/rancher/scc-operator/internal/telemetry"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"

	"github.com/rancher/scc-operator/internal/consts"
	"github.com/rancher/scc-operator/internal/initializer"
	"github.com/rancher/scc-operator/internal/repos/secretrepo"
	"github.com/rancher/scc-operator/internal/suseconnect"
	"github.com/rancher/scc-operator/internal/suseconnect/credentials"
	"github.com/rancher/scc-operator/internal/suseconnect/offline"
	"github.com/rancher/scc-operator/internal/types"
	wranglerPolyfill "github.com/rancher/scc-operator/internal/wrangler/polyfill"
	v1 "github.com/rancher/scc-operator/pkg/apis/scc.cattle.io/v1"
	"github.com/rancher/scc-operator/pkg/controllers/helpers"
	"github.com/rancher/scc-operator/pkg/controllers/shared"
	registrationControllers "github.com/rancher/scc-operator/pkg/generated/controllers/scc.cattle.io/v1"
	"github.com/rancher/scc-operator/pkg/util/log"
)

const (
	controllerID    = "prime-registration"
	prodBaseCheckin = time.Hour * 20
	prodMinCheckin  = prodBaseCheckin - (3 * time.Hour)
	devBaseCheckin  = time.Minute * 30
	devMinCheckin   = devBaseCheckin - (10 * time.Minute)
)

// SCCHandler Defines a common interface for online and offline operations
// IMPORTANT: All the `Reconcile*` methods modifies the object in memory but does NOT save it. The caller is responsible for saving the state.
type SCCHandler interface {
	// SetRancherMetrics adds the current Rancher metrics to the SCCHandler for processing
	// This must be added after initialization so that we only fetch metrics if we need to sync.
	SetRancherMetrics(rancherMetrics telemetry.MetricsWrapper)
	// NeedsRegistration determines if the system requires initial SCC registration.
	NeedsRegistration(*v1.Registration) bool
	// NeedsActivation checks if the system requires activation with SCC.
	NeedsActivation(*v1.Registration) bool
	// ReadyForActivation checks if the system is ready for activation.
	ReadyForActivation(*v1.Registration) bool
	// ResetToRegisteredForActivation will clean up the registration back to the ReadyForActivation state
	ResetToRegisteredForActivation(*v1.Registration) (*v1.Registration, error)

	// PrepareForRegister preforms pre-registration steps
	PrepareForRegister(*v1.Registration) (*v1.Registration, error)
	// Register performs the initial system registration with SCC or creates an offline request.
	Register(*v1.Registration) (suseconnect.RegistrationSystemID, error)
	// PrepareRegisteredForActivation prepares the Registration object after successful registration.
	PrepareRegisteredForActivation(*v1.Registration) (*v1.Registration, error)
	// Activate activates the system with SCC or verifies an offline request.
	Activate(*v1.Registration) error
	// PrepareActivatedForKeepalive prepares an Activated Registration for future keepalive
	PrepareActivatedForKeepalive(*v1.Registration) (*v1.Registration, error)
	// Keepalive provides a heartbeat to SCC and validates the system's status.
	Keepalive(registrationObj *v1.Registration) error
	// PrepareKeepaliveSucceeded completes any necessary steps after successful keepalive
	PrepareKeepaliveSucceeded(*v1.Registration) (*v1.Registration, error)
	// Deregister initiates the system's deregistration from SCC.
	Deregister() error

	// ReconcileRegisterError prepares the Registration object for error reconciliation after RegisterSystem fails.
	ReconcileRegisterError(*v1.Registration, error, types.RegistrationPhase) *v1.Registration
	// ReconcileKeepaliveError prepares the Registration object for error reconciliation after Keepalive fails.
	ReconcileKeepaliveError(*v1.Registration, error) *v1.Registration
	// ReconcileActivateError prepares the Registration object for error reconciliation after Activate fails.
	ReconcileActivateError(*v1.Registration, error, types.ActivationPhase) *v1.Registration
}

type handler struct {
	ctx               context.Context
	log               *logrus.Entry
	options           *types.RunOptions
	registrations     registrationControllers.RegistrationController
	registrationCache registrationControllers.RegistrationCache
	secretRepo        *secretrepo.SecretRepository
	settings          *settings.SettingReader
}

// Register will setup the SCC registration CRDs controllers (and related secret controllers)
// TODO: pull out secret stuff to their own controller
func Register(
	ctx context.Context,
	options *types.RunOptions,
	registrations registrationControllers.RegistrationController,
	secretsRepo *secretrepo.SecretRepository,
	settings *settings.SettingReader,
) {
	controller := &handler{
		log:               log.NewControllerLogger("registration-controller"),
		ctx:               ctx,
		options:           options,
		registrations:     registrations,
		registrationCache: registrations.Cache(),
		secretRepo:        secretsRepo,
		settings:          settings,
	}

	controller.initIndexers()
	controller.initResolvers(ctx)

	withinExpectedNamespaceCondition := func(name string, obj runtime.Object) (bool, error) {
		if !wranglerPolyfill.InExpectedNamespace(name, obj, controller.options.SystemNamespace) {
			return false, nil
		}
		return true, nil
	}
	wranglerPolyfill.ScopedOnChange(ctx, controllerID+"-secrets", withinExpectedNamespaceCondition, secretsRepo.Controller, controller.OnSecretChange)
	wranglerPolyfill.ScopedOnRemove(ctx, controllerID+"-secrets-remove", withinExpectedNamespaceCondition, secretsRepo.Controller, controller.OnSecretRemove)

	// TODO: pull out registration controllers to register only when system is ready
	// TODO: also add a watcher to trigger enqueue on related resource changes
	withinOperatorScopeCondition := func(_ string, obj runtime.Object) (bool, error) {
		if obj == nil {
			return false, nil
		}
		metaObj, err := meta.Accessor(obj)
		if err != nil {
			return false, err
		}

		return helpers.ShouldManage(metaObj, controller.options.OperatorName), nil
	}
	wranglerPolyfill.ScopedOnChange(ctx, controllerID, withinOperatorScopeCondition, registrations, controller.OnRegistrationChange)
	wranglerPolyfill.ScopedOnRemove(ctx, controllerID+"-remove", withinOperatorScopeCondition, registrations, controller.OnRegistrationRemove)

	cfg := setupCfg()
	go controller.RunLifecycleManager(cfg, rancher.GetServerURL(ctx, settings))
}

func (h *handler) prepareHandler(registrationObj *v1.Registration, rancherURL string) SCCHandler {
	ref := registrationObj.ToOwnerRef()
	nameSuffixHash := registrationObj.Labels[consts.LabelNameSuffix]

	defaultLabels := map[string]string{
		consts.LabelSccHash:      registrationObj.Labels[consts.LabelSccHash],
		consts.LabelNameSuffix:   nameSuffixHash,
		consts.LabelSccManagedBy: controllerID,
		consts.LabelK8sManagedBy: initializer.OperatorName.Get(),
	}

	if registrationObj.Spec.Mode == v1.RegistrationModeOffline {
		offlineRequestSecretName := consts.OfflineRequestSecretName(nameSuffixHash)
		offlineCertSecretName := consts.OfflineCertificateSecretName(nameSuffixHash)
		return &sccOfflineMode{
			rancherURL:   rancherURL,
			rancherUUID:  rancher.GetRancherInstallUUID(h.ctx, h.settings),
			log:          h.log.WithField("regHandler", "offline"),
			options:      h.options,
			registration: registrationObj,
			offlineSecrets: offline.New(
				h.options.SystemNamespace,
				offlineRequestSecretName,
				offlineCertSecretName,
				ref,
				h.secretRepo,
				defaultLabels,
			),
		}
	}

	credsSecretName := consts.SCCCredentialsSecretName(nameSuffixHash)
	return &sccOnlineMode{
		rancherURL:   rancherURL,
		log:          h.log.WithField("regHandler", "online"),
		options:      h.options,
		registration: registrationObj,
		sccCredentials: credentials.New(
			h.options.SystemNamespace,
			credsSecretName,
			ref,
			h.secretRepo,
			defaultLabels,
		),
		secretRepo: h.secretRepo,
	}
}

func (h *handler) OnSecretChange(_ string, incomingObj *corev1.Secret) (*corev1.Secret, error) {
	if incomingObj == nil || incomingObj.DeletionTimestamp != nil {
		return incomingObj, nil
	}

	if !h.isSCCEntrypointSecret(incomingObj) {
		return incomingObj, nil
	}

	// TODO(dan): sort out this to validate logic more
	// TODO: needs something to handle adopting new and unowned instances?
	if !helpers.ShouldManage(incomingObj, h.options.OperatorName) {
		// When the secret has no managedBy label, we should assume ownership I guess?
		if !helpers.HasManagedByLabel(incomingObj) {
			h.log.Debugf("taking ownership of the unowned entrypoint secret")
			prepared := incomingObj.DeepCopy()
			prepared = helpers.TakeOwnership(prepared, h.options.OperatorName)
			_, updateErr := h.secretRepo.RetryingPatchUpdate(incomingObj, prepared)
			if updateErr != nil {
				h.log.Errorf("failed to take ownership of secret %s/%s: %v", incomingObj.Namespace, incomingObj.Name, updateErr)
				return incomingObj, updateErr
			}

			h.log.Debugf("Secret %s/%s is now managed by %s", incomingObj.Namespace, incomingObj.Name, h.options.OperatorName)

			return incomingObj, nil
		}

		h.log.Debugf("Secret %s/%s is not managed by %s, skipping", incomingObj.Namespace, incomingObj.Name, h.options.OperatorName)
		return incomingObj, nil
	}

	if _, saltOk := incomingObj.GetLabels()[consts.LabelObjectSalt]; !saltOk {
		return h.prepareSecretSalt(incomingObj)
	}

	incomingNameHash := incomingObj.GetLabels()[consts.LabelNameSuffix]
	incomingContentHash := incomingObj.GetLabels()[consts.LabelSccHash]
	params, err := extractRegistrationParamsFromSecret(incomingObj, h.options.OperatorName)
	if err != nil {
		return incomingObj, fmt.Errorf("failed to extract registration params from secret %s/%s: %w", incomingObj.Namespace, incomingObj.Name, err)
	}

	if incomingContentHash == "" {
		h.log.Debugf("incoming content hash empty, preparing secret %s/%s", incomingObj.Namespace, incomingObj.Name)
		// update secret with useful annotations & labels
		newSecret := incomingObj.DeepCopy()
		if newSecret.Annotations == nil {
			newSecret.Annotations = map[string]string{}
		}
		newSecret.Annotations[consts.LabelSccLastProcessed] = time.Now().Format(time.RFC3339)
		maps.Copy(newSecret.Labels, params.Labels())

		_, updateErr := h.secretRepo.RetryingPatchUpdate(incomingObj, newSecret)
		if updateErr != nil {
			h.log.Error("error applying metadata updates to default SCC registration secret")
			return nil, updateErr
		}

		return incomingObj, nil
	}

	// If secret hash has changed make sure that we submit objects that correspond to that hash
	// are cleaned up
	// TODO: make it so that changes to the incoming Salt (which changes the nameID) are correctly handled
	// Note that change would affect both name and content hashes - however something seems to not.
	if incomingNameHash != params.nameID {
		h.log.Info("must cleanup existing registration managed by secret")
		if cleanUpErr := h.cleanupRegistrationByHash(hashCleanupRequest{
			incomingNameHash,
			NameHash,
		}); cleanUpErr != nil {
			h.log.Errorf("failed to cleanup registrations for hash %s: %v", incomingNameHash, cleanUpErr)
			return incomingObj, cleanUpErr
		}
	}

	// TODO: rework stuff around this as this shouldn't be necessary
	if incomingContentHash != params.contentHash {
		h.log.Info("must cleanup existing registration managed by secret")
		if cleanUpErr := h.cleanupRelatedSecretsByHash(incomingContentHash); cleanUpErr != nil {
			h.log.Errorf("failed to cleanup registrations for hash %s: %v", incomingNameHash, cleanUpErr)
			return incomingObj, cleanUpErr
		}
	}

	h.log.Info("create or update registration managed by secret")

	// update secret with useful annotations & labels
	newSecret := incomingObj.DeepCopy()
	if newSecret.Annotations == nil {
		newSecret.Annotations = map[string]string{}
	}
	newSecret.Annotations[consts.LabelSccLastProcessed] = time.Now().Format(time.RFC3339)

	labels := incomingObj.Labels
	maps.Copy(labels, params.Labels())
	newSecret.Labels = labels

	if _, err := h.secretRepo.RetryingPatchUpdate(incomingObj, newSecret); err != nil {
		return incomingObj, err
	}

	if params.regType == v1.RegistrationModeOffline && params.hasOfflineCertData {
		offlineCertSecret, err := h.offlineCertFromSecretEntrypoint(params)
		if err != nil {
			return incomingObj, err
		}

		if _, err := h.secretRepo.CreateOrUpdateSecret(offlineCertSecret); err != nil {
			return incomingObj, err
		}
	}

	if params.regType == v1.RegistrationModeOnline {
		regCodeSecret, err := h.regCodeFromSecretEntrypoint(params)
		if err != nil {
			return incomingObj, err
		}

		if _, err := h.secretRepo.CreateOrUpdateSecret(regCodeSecret); err != nil {
			return incomingObj, err
		}
	}

	// construct associated registration CRs
	registration, err := h.registrationFromSecretEntrypoint(params)
	if err != nil {
		return incomingObj, fmt.Errorf("failed to create registration from secret %s/%s: %w", incomingObj.Namespace, incomingObj.Name, err)
	}

	if createOrUpdateErr := h.createOrUpdateRegistration(registration); createOrUpdateErr != nil {
		h.log.Errorf("failed to create or update registration %s: %v", registration.Name, createOrUpdateErr)
		return incomingObj, fmt.Errorf("failed to create or update registration %s: %w", registration.Name, createOrUpdateErr)
	}

	return incomingObj, nil
}

func (h *handler) cleanupRegistrationByHash(cleanupRequest hashCleanupRequest) error {
	var regs []*v1.Registration
	var err error
	if cleanupRequest.hashType == ContentHash {
		regs, err = h.registrationCache.GetByIndex(IndexRegistrationsBySccHash, cleanupRequest.hash)
	} else {
		regs, err = h.registrationCache.GetByIndex(IndexRegistrationsByNameHash, cleanupRequest.hash)
	}
	if err != nil {
		return err
	}

	h.log.Infof("found %d matching registrations to clean up the %s hash", len(regs), cleanupRequest.hashType)

	for _, reg := range regs {
		if !slices.Contains(reg.Finalizers, consts.FinalizerSccRegistration) {
			continue
		}

		if retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			var remainingFin []string
			for _, finalizer := range reg.Finalizers {
				if finalizer != consts.FinalizerSccRegistration {
					remainingFin = append(remainingFin, finalizer)
				}
			}
			reg.Finalizers = remainingFin

			_, updateErr := h.registrations.Update(reg)
			return updateErr
		}); retryErr != nil {
			return retryErr
		}

		deleteErr := h.registrations.Delete(reg.Name, &metav1.DeleteOptions{})
		if apierrors.IsNotFound(deleteErr) {
			h.log.Debugf("Registration %s already deleted", reg.Name)
			continue
		}
		if deleteErr != nil {
			return fmt.Errorf("failed to delete registration %s: %w", reg.Name, deleteErr)
		}
	}
	return nil
}

func (h *handler) cleanupRelatedSecretsByHash(contentHash string) error {
	secrets, err := h.secretRepo.GetBySccContentHash(contentHash)
	h.log.Infof("found %d matching related secrets to clean up; content hash of %s", len(secrets), contentHash)
	if err != nil {
		return err
	}

	// It should never be in there, but just in case don't act on the entrypoint
	secrets = slices.Collect(func(yield func(secret *corev1.Secret) bool) {
		for _, secret := range secrets {
			if secret.Name != consts.ResourceSCCEntrypointSecretName && !strings.HasPrefix(secret.Name, consts.OfflineRequestSecretNamePrefix) {
				if !yield(secret) {
					return
				}
			}
		}
	})

	for _, secret := range secrets {
		if shared.SecretHasCredentialsFinalizer(secret) ||
			shared.SecretHasRegCodeFinalizer(secret) {

			var updateErr error
			secretUpdated := secret.DeepCopy()
			secretUpdated = shared.SecretRemoveCredentialsFinalizer(secretUpdated)
			secretUpdated = shared.SecretRemoveRegCodeFinalizer(secretUpdated)
			_, updateErr = h.secretRepo.RetryingPatchUpdate(secret, secretUpdated)
			if updateErr != nil {
				h.log.Errorf("failed to update secret %s/%s: %v", secret.Namespace, secret.Name, updateErr)
				return updateErr
			}
		}

		deleteErr := h.secretRepo.Controller.Delete(secret.Namespace, secret.Name, &metav1.DeleteOptions{})
		if apierrors.IsNotFound(deleteErr) {
			h.log.Debugf("Related Secret %s/%s already deleted", secret.Namespace, secret.Name)
			continue
		}
		if deleteErr != nil {
			return fmt.Errorf("failed to delete secret %s/%s: %w", secret.Namespace, secret.Name, deleteErr)
		}
	}

	return nil
}

func (h *handler) OnSecretRemove(_ string, incomingObj *corev1.Secret) (*corev1.Secret, error) {
	if incomingObj == nil {
		return nil, nil
	}
	if incomingObj.Namespace != h.options.SystemNamespace {
		h.log.Debugf("Secret %s/%s is not in SCC system namespace %s, skipping cleanup", incomingObj.Namespace, incomingObj.Name, h.options.SystemNamespace)
		return incomingObj, nil
	}

	if !helpers.ShouldManage(incomingObj, h.options.OperatorName) {
		h.log.Debugf("Secret %s/%s is not managed by %s, skipping", incomingObj.Namespace, incomingObj.Name, h.options.OperatorName)
		return incomingObj, nil
	}

	if h.isSCCEntrypointSecret(incomingObj) {
		hash, ok := incomingObj.Labels[consts.LabelNameSuffix]
		if !ok {
			return incomingObj, nil
		}

		// TODO: (alex) needs some thought about how we actually map entrypoint secret cleanup
		// here based on the control flow changes in OnChange
		if err := h.cleanupRegistrationByHash(hashCleanupRequest{
			hash,
			NameHash,
		}); err != nil {
			h.log.Errorf("failed to cleanup registrations for hash %s: %v", hash, err)
			return nil, err
		}
		contentHash, ok := incomingObj.Labels[consts.LabelSccHash]
		if !ok {
			return incomingObj, nil
		}
		if cleanUpErr := h.cleanupRelatedSecretsByHash(contentHash); cleanUpErr != nil {
			h.log.Errorf("failed to cleanup registrations for hash %s: %v", hash, cleanUpErr)
			return incomingObj, cleanUpErr
		}

		return incomingObj, nil
	}

	if shared.SecretHasCredentialsFinalizer(incomingObj) ||
		shared.SecretHasRegCodeFinalizer(incomingObj) {
		refs := incomingObj.GetOwnerReferences()
		danglingRefs := 0
		for _, ref := range refs {
			if ref.APIVersion == v1.SchemeGroupVersion.String() &&
				ref.Kind == "Registration" {
				reg, err := h.registrations.Get(ref.Name, metav1.GetOptions{})
				if apierrors.IsNotFound(err) {
					continue
				}

				if reg.DeletionTimestamp == nil && reg.Status.ActivationStatus.Activated {
					danglingRefs++
				} else {
					// TODO(alex): verify this logic when you are back
					// When reg is marked to delete too - we may need to help clean it up
					regFinalizers := reg.GetFinalizers()
					if len(regFinalizers) > 0 && slices.Contains(regFinalizers, consts.FinalizerSccRegistration) {
						regUpdate := reg.DeepCopy()
						removeIndex := slices.Index(regFinalizers, consts.FinalizerSccRegistration)
						regUpdate.Finalizers = append(reg.Finalizers[:removeIndex], reg.Finalizers[removeIndex+1:]...)
						_, err = h.patchUpdateRegistration(reg, regUpdate)
						if err != nil {
							h.log.Errorf("failed to patch registration %s/%s: %v", reg.Namespace, reg.Name, err)
						}
					}
				}
			}
		}
		if danglingRefs > 0 {
			h.log.Errorf("cannot remove SCC finalizer from secret %s/%s, dangling references to Registration found", incomingObj.Namespace, incomingObj.Name)
			return nil, fmt.Errorf("cannot remove SCC finalizer from secret %s/%s, dangling references to Registration found", incomingObj.Namespace, incomingObj.Name)
		}
		newSecret := incomingObj.DeepCopy()
		if shared.SecretHasCredentialsFinalizer(newSecret) {
			newSecret = shared.SecretRemoveCredentialsFinalizer(newSecret)
		}
		if shared.SecretHasRegCodeFinalizer(newSecret) {
			newSecret = shared.SecretRemoveRegCodeFinalizer(newSecret)
		}
		logrus.Info("Removing finalizer from secret", newSecret.Name, "in namespace", newSecret.Namespace)
		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			_, err := h.secretRepo.PatchUpdate(incomingObj, newSecret)
			return err
		}); err != nil {
			h.log.Errorf("failed to remove SCC finalizer from secret %s/%s: %v", incomingObj.Namespace, incomingObj.Name, err)
			return nil, fmt.Errorf("failed to remove SCC finalizer from secret %s/%s: %w", incomingObj.Namespace, incomingObj.Name, err)
		}
	}

	return incomingObj, nil
}

func (h *handler) OnRegistrationChange(_ string, registrationObj *v1.Registration) (*v1.Registration, error) {
	activiateMu.Lock()
	defer activiateMu.Unlock()
	if registrationObj == nil || registrationObj.DeletionTimestamp != nil {
		return nil, nil
	}

	rancherURL := rancher.GetServerURL(h.ctx, h.settings)
	if rancherURL == "" {
		h.log.Info("Server URL not set")
		return registrationObj, errors.New("no server url found in the system info")
	}

	registrationHandler := h.prepareHandler(registrationObj, rancherURL)

	if registrationObj.Spec.Mode == v1.RegistrationModeOffline {
		if v1.ResourceConditionFailure.IsTrue(registrationObj) && v1.RegistrationConditionOfflineCertificateReady.IsFalse(registrationObj) && v1.RegistrationConditionOfflineCertificateReady.GetMessage(registrationObj) != InitialOfflineCertificateReadyMessage && registrationObj.Spec.OfflineRegistrationCertificateSecretRef == nil {
			h.log.Info("registration is failed but user removed certificate. Resetting registration status back to ReadyForActivation")

			resetUpdateErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				reset := registrationObj.DeepCopy()

				reset, resetErr := registrationHandler.ResetToRegisteredForActivation(reset)
				if resetErr != nil {
					return resetErr
				}

				_, updateErr := h.registrations.UpdateStatus(reset)
				return updateErr
			})
			if resetUpdateErr != nil {
				return registrationObj, resetUpdateErr
			}

			return registrationObj, nil
		}
	}

	if shared.RegistrationIsFailed(registrationObj) {
		failedCondition := registrationObj.Status.CurrentCondition
		h.log.Errorf("registration `%s` has the Failure status condition from: %v", registrationObj.Name, failedCondition)
		h.log.Warnf("reviewing the registration `%s` for other errors is advised before retrying", registrationObj.Name)

		errorFixHint := fmt.Sprintf("delete this registration `%s` and then create a new one to try again.", registrationObj.Name)
		if shared.RegistrationHasManagedFinalizer(registrationObj) {
			errorFixHint = fmt.Sprintf("delete the entrypoint secret `%s/%s`, give it time to clean up, and then create a new one to try again.", h.options.SystemNamespace, consts.ResourceSCCEntrypointSecretName)
		}
		h.log.Warn("after resolving the issue(s), " + errorFixHint)
		return registrationObj, nil
	}

	// Skip keepalive for anything activated within the last 20 hours
	if !registrationHandler.NeedsRegistration(registrationObj) &&
		!registrationHandler.NeedsActivation(registrationObj) &&
		registrationObj.Spec.SyncNow == nil {
		if !registrationObj.Status.ActivationStatus.LastValidatedTS.IsZero() &&
			registrationObj.Status.ActivationStatus.LastValidatedTS.Time.After(minResyncInterval()) {
			return registrationObj, nil
		}
	}

	// Fetch Rancher metrics for SCC
	systemMetrics, metricsErr := h.secretRepo.FetchMetricsSecret()
	if metricsErr != nil {
		wrappedErr := fmt.Errorf("encountered additional error when preparing SCC handler: %v", metricsErr)
		h.log.Error(wrappedErr)
		return registrationObj, wrappedErr
	}
	// TODO: parse out the secret data
	registrationHandler.SetRancherMetrics(systemMetrics)

	// Only on the first time an object passes through here should it need to be registered
	// The logical default condition should always be to try activation, unless we know it's not registered.
	if registrationHandler.NeedsRegistration(registrationObj) {
		if !registrationObj.HasCondition(v1.ResourceConditionProgressing) || v1.ResourceConditionProgressing.IsFalse(registrationObj) {
			progressingObj := registrationObj.DeepCopy()
			// Set object to progressing
			progressingUpdateErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				var err error
				v1.ResourceConditionProgressing.True(progressingObj)
				// Set ResourceConditionProgressing as the CurrentCondition since we're starting the registration process
				progressingObj.SetCurrentCondition(v1.ResourceConditionProgressing)
				progressingObj, err = h.registrations.UpdateStatus(progressingObj)
				return err
			})
			if progressingUpdateErr != nil {
				return registrationObj, progressingUpdateErr
			}

			return registrationObj, nil
		}

		// Start of initial registration/announce of cluster
		regForAnnounce := registrationObj.DeepCopy()
		preparedForRegister, prepareErr := registrationHandler.PrepareForRegister(regForAnnounce)
		if prepareErr != nil {
			err := h.reconcileRegistration(registrationHandler, preparedForRegister, prepareErr, types.RegistrationPrepare)
			return registrationObj, err
		}

		var updateErr error
		if regForAnnounce, updateErr = h.registrations.UpdateStatus(preparedForRegister); updateErr != nil {
			return registrationObj, updateErr
		}

		announcedSystemID, registerErr := registrationHandler.Register(regForAnnounce)
		if registerErr != nil {
			err := h.reconcileRegistration(registrationHandler, preparedForRegister, registerErr, types.RegistrationMain)
			return registrationObj, err
		}

		setSystemID := false
		switch announcedSystemID {
		case suseconnect.OfflineRegistrationSystemID:
			h.log.Debugf("SCC system ID cannot be known for offline until activation")
		case suseconnect.KeepAliveRegistrationSystemID:
			h.log.Debugf("register system handled via keepalive")
			announcedSystemID = suseconnect.RegistrationSystemID(*registrationObj.Status.SCCSystemID)
		default:
			h.log.Debugf("Annoucned System ID: %v", announcedSystemID)
			setSystemID = true
		}

		var prepareError error
		// Prepare the Registration for Activation phase
		if setSystemID {
			regForAnnounce.Status.SCCSystemID = announcedSystemID.Ptr()
		}
		regForAnnounce, prepareError = registrationHandler.PrepareRegisteredForActivation(regForAnnounce)
		if prepareError != nil {
			err := h.reconcileRegistration(registrationHandler, preparedForRegister, prepareError, types.RegistrationForActivation)
			return registrationObj, err
		}
		regForAnnounce.Status.RegistrationProcessedTS = &metav1.Time{
			Time: time.Now(),
		}

		_, registerUpdateErr := h.registrations.UpdateStatus(regForAnnounce)
		if registerUpdateErr != nil {
			return registrationObj, registerUpdateErr
		}

		return registrationObj, nil
	}

	if registrationHandler.NeedsActivation(registrationObj) {
		if !registrationHandler.ReadyForActivation(registrationObj) {
			h.log.Debugf("registration needs to be activated, but not yet ready; %v", registrationObj)
			return registrationObj, nil
		}
		activationErr := registrationHandler.Activate(registrationObj)
		// reconcile error state - must be able to handle Auth errors (or other SCC sourced errors)
		if activationErr != nil {
			err := h.reconcileActivation(registrationHandler, registrationObj, activationErr, types.ActivationMain)
			return registrationObj, err
		}

		activatedUpdateErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			var retryErr, updateErr error
			registrationObj, retryErr = h.registrations.Get(registrationObj.Name, metav1.GetOptions{})
			if retryErr != nil {
				return retryErr
			}

			activated := registrationObj.DeepCopy()
			activated = shared.PrepareSuccessfulActivation(activated)
			prepared, err := registrationHandler.PrepareActivatedForKeepalive(activated)
			if err != nil {
				err := h.reconcileActivation(registrationHandler, registrationObj, activationErr, types.ActivationPrepForKeepalive)
				return err
			}
			_, updateErr = h.registrations.UpdateStatus(prepared)
			return updateErr
		})
		if activatedUpdateErr != nil {
			return registrationObj, activatedUpdateErr
		}

		return registrationObj, nil
	}

	// Handle what to do when CheckNow is used...
	if shared.RegistrationNeedsSyncNow(registrationObj) {
		if registrationObj.Spec.Mode == v1.RegistrationModeOffline {
			updated := registrationObj.DeepCopy()
			updated.Spec = *registrationObj.Spec.WithoutSyncNow()

			offlineHandler := registrationHandler.(*sccOfflineMode)
			refreshErr := offlineHandler.RefreshOfflineRequestSecret()
			_, updateErr := h.registrations.Update(updated)
			if updateErr != nil || refreshErr != nil {
				return registrationObj, errors.Join(refreshErr, updateErr)
			}

			return registrationObj, nil
		}

		// Todo: online/offline handler interface should have a SyncNow call to get rid of the if here
		updated := registrationObj.DeepCopy()
		updated.Spec = *registrationObj.Spec.WithoutSyncNow()
		updated.Status.ActivationStatus.Activated = false
		updated.Status.ActivationStatus.LastValidatedTS = &metav1.Time{}
		v1.ResourceConditionProgressing.True(updated)
		v1.ResourceConditionReady.False(updated)
		v1.ResourceConditionDone.False(updated)
		v1.RegistrationConditionActivated.False(updated)
		// Set ResourceConditionProgressing as the CurrentCondition since we're resetting the registration process
		updated.SetCurrentCondition(v1.ResourceConditionProgressing)

		var err error
		updated, err = h.registrations.UpdateStatus(updated)
		if err != nil {
			// TODO handle this error better via ReconcileSyncNow
			return registrationObj, err
		}

		updated.Spec = *registrationObj.Spec.WithoutSyncNow()
		updated, err = h.registrations.Update(updated)
		return registrationObj, err
	}

	keepaliveErr := registrationHandler.Keepalive(registrationObj)
	if keepaliveErr != nil {
		reconcileErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			curReg, getErr := h.registrations.Get(registrationObj.Name, metav1.GetOptions{})
			if getErr != nil {
				return getErr
			}

			prepareObj := curReg.DeepCopy()
			prepareObj = registrationHandler.ReconcileKeepaliveError(prepareObj, keepaliveErr)

			_, reconcileUpdateErr := h.registrations.Update(prepareObj)
			return reconcileUpdateErr
		})

		err := fmt.Errorf("keepalive failed: %w", keepaliveErr)
		if reconcileErr != nil {
			err = fmt.Errorf("keepalive failed with additional errors: %w, %w", keepaliveErr, reconcileErr)
		}

		return registrationObj, err
	}

	keepaliveUpdateErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var retryErr, updateErr error
		registrationObj, retryErr = h.registrations.Get(registrationObj.Name, metav1.GetOptions{})
		if retryErr != nil {
			return retryErr
		}

		keepalive := registrationObj.DeepCopy()
		keepalive = shared.PrepareSuccessfulActivation(keepalive)
		v1.RegistrationConditionKeepalive.True(keepalive)
		prepared, err := registrationHandler.PrepareKeepaliveSucceeded(keepalive)
		if err != nil {
			return err
		}
		_, updateErr = h.registrations.UpdateStatus(prepared)
		return updateErr
	})
	if keepaliveUpdateErr != nil {
		return registrationObj, keepaliveUpdateErr
	}

	return registrationObj, nil
}

func (h *handler) OnRegistrationRemove(name string, registrationObj *v1.Registration) (*v1.Registration, error) {
	if registrationObj == nil {
		return nil, nil
	}

	rancherURL := rancher.GetServerURL(h.ctx, h.settings)
	if rancherURL == "" {
		h.log.Info("Server URL not set")
		return registrationObj, errors.New("no server url found in the system info")
	}
	regHandler := h.prepareHandler(registrationObj, rancherURL)
	deRegErr := regHandler.Deregister()
	if deRegErr != nil {
		h.log.Warn(deRegErr)
	}

	err := h.registrations.Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return registrationObj, err
	}

	return nil, nil
}
