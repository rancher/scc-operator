package controllers

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"maps"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rancher/scc-operator/internal/consts"
	coreUtil "github.com/rancher/scc-operator/internal/initializer"
	v1 "github.com/rancher/scc-operator/pkg/apis/scc.cattle.io/v1"
	"github.com/rancher/scc-operator/pkg/controllers/shared"
	"github.com/rancher/scc-operator/pkg/util"
	"github.com/rancher/scc-operator/pkg/util/log"
	"github.com/rancher/scc-operator/pkg/util/salt"
)

type HashType int

const (
	NameHash HashType = iota
	ContentHash
)

func (ht *HashType) String() string {
	if *ht == NameHash {
		return "name"
	}

	return "content"
}

type hashCleanupRequest struct {
	hash     string
	hashType any
}

const (
	dataKeyRegistrationType = "registrationType"
)

func (h *handler) isSCCEntrypointSecret(secretObj *corev1.Secret) bool {
	if secretObj.Name != consts.ResourceSCCEntrypointSecretName || secretObj.Namespace != h.options.SystemNamespace {
		return false
	}
	return true
}

// prepareSecretSalt applies an instance salt onto an entrypoint secret used to create randomness in hashes
func (h *handler) prepareSecretSalt(secret *corev1.Secret) (*corev1.Secret, error) {
	preparedSecret := secret.DeepCopy()
	generatedSalt := salt.NewSaltGen(nil, nil).GenerateSalt()

	existingLabels := make(map[string]string)
	if objLabels := secret.GetLabels(); objLabels != nil {
		existingLabels = objLabels
	}
	existingLabels[consts.LabelObjectSalt] = generatedSalt
	preparedSecret.SetLabels(existingLabels)

	_, updateErr := h.secretRepo.RetryingPatchUpdate(secret, preparedSecret)
	if updateErr != nil {
		h.log.Error("error applying metadata updates to default SCC registration secret; cannot initialize secret salt value")
		return nil, updateErr
	}

	return secret, nil
}

func getCurrentRegURL(secret *corev1.Secret) (regURL []byte) {
	regURLBytes, ok := secret.Data[consts.RegistrationURL]
	if ok {
		return regURLBytes
	}
	if util.HasGlobalPrimeRegistrationURL() {
		globalRegistrationURL := util.GetGlobalPrimeRegistrationURL()
		return []byte(globalRegistrationURL)
	}
	if coreUtil.DevMode.Get() {
		return []byte(consts.StagingSccURL)
	}
	return []byte{}
}

func extractRegistrationParamsFromSecret(secret *corev1.Secret, managedBy string) (RegistrationParams, error) {
	incomingSalt := []byte(secret.GetLabels()[consts.LabelObjectSalt])

	regMode := v1.RegistrationModeOnline
	regType, ok := secret.Data[dataKeyRegistrationType]
	if !ok || len(regType) == 0 {
		// TODO: consider using an existing logger
		log.NewLog().Warnf("secret does not have the `%s` field, defaulting to %s", dataKeyRegistrationType, regMode)
	} else {
		regMode = v1.RegistrationMode(regType)
		if !regMode.Valid() {
			return RegistrationParams{}, fmt.Errorf("invalid registration mode %s", string(regMode))
		}
	}

	regCode, ok := secret.Data[consts.SecretKeyRegistrationCode]
	if !ok || len(regCode) == 0 {
		if regMode == v1.RegistrationModeOnline {
			return RegistrationParams{}, fmt.Errorf("secret does not have data %s; this is required in online mode", consts.SecretKeyRegistrationCode)
		}
	}

	offlineRegCertData, certOk := secret.Data[consts.SecretKeyOfflineRegCert]
	hasOfflineCert := certOk && len(offlineRegCertData) > 0

	var regURLBytes []byte
	regURLString := ""
	if regMode == v1.RegistrationModeOnline {
		regURLBytes = getCurrentRegURL(secret)
		regURLString = string(regURLBytes)
	}

	hasher := md5.New()
	nameData := append(incomingSalt, regType...)
	nameData = append(nameData, regCode...)
	nameData = append(nameData, regURLBytes...)
	data := append(nameData, offlineRegCertData...)

	// Generate a hash for the name data
	if _, err := hasher.Write(nameData); err != nil {
		return RegistrationParams{}, fmt.Errorf("failed to hash name data: %v", err)
	}
	nameID := hex.EncodeToString(hasher.Sum(nil))

	// Generate hash for the content data
	if _, err := hasher.Write(data); err != nil {
		return RegistrationParams{}, fmt.Errorf("failed to hash data: %v", err)
	}
	contentsID := hex.EncodeToString(hasher.Sum(nil))

	return RegistrationParams{
		managedByOperator: managedBy,
		regType:           regMode,
		nameID:            nameID,
		contentHash:       contentsID,
		regCode:           regCode,
		regCodeSecretRef: &corev1.SecretReference{
			Name:      consts.RegistrationCodeSecretName(nameID),
			Namespace: secret.Namespace,
		},
		hasOfflineCertData: hasOfflineCert,
		offlineCertData:    &offlineRegCertData,
		offlineCertSecretRef: &corev1.SecretReference{
			Name:      consts.OfflineCertificateSecretName(nameID),
			Namespace: secret.Namespace,
		},
		regURL: regURLString,
	}, nil
}

type RegistrationParams struct {
	managedByOperator    string
	regType              v1.RegistrationMode
	nameID               string
	contentHash          string
	regCode              []byte
	regCodeSecretRef     *corev1.SecretReference
	regURL               string
	hasOfflineCertData   bool
	offlineCertData      *[]byte
	offlineCertSecretRef *corev1.SecretReference
}

func (r RegistrationParams) Labels() map[string]string {
	return map[string]string{
		consts.LabelNameSuffix:   r.nameID,
		consts.LabelSccHash:      r.contentHash,
		consts.LabelSccManagedBy: consts.ManagedByValueSecretBroker,
		consts.LabelK8sManagedBy: r.managedByOperator,
	}
}

func (h *handler) registrationFromSecretEntrypoint(
	params RegistrationParams,
) (*v1.Registration, error) {
	if !params.regType.Valid() {
		return nil, fmt.Errorf(
			"invalid registration type %s, must be one of %s or %s",
			params.regType,
			v1.RegistrationModeOnline,
			v1.RegistrationModeOffline,
		)
	}

	hashedName := fmt.Sprintf("scc-registration-%s", params.nameID)
	var reg *v1.Registration
	var err error

	reg, err = h.registrationCache.Get(hashedName)
	if err != nil && apierrors.IsNotFound(err) {
		reg = &v1.Registration{
			ObjectMeta: metav1.ObjectMeta{
				Name: hashedName,
			},
		}
	}
	if reg.Labels == nil {
		reg.Labels = map[string]string{}
	}
	maps.Copy(reg.Labels, params.Labels())

	reg.Spec = paramsToRegSpec(params)
	if !shared.RegistrationHasManagedFinalizer(reg) {
		reg = shared.RegistrationAddManagedFinalizer(reg)
	}

	return reg, nil
}

func paramsToRegSpec(params RegistrationParams) v1.RegistrationSpec {
	regSpec := v1.RegistrationSpec{
		Mode: params.regType,
	}

	if params.regType == v1.RegistrationModeOnline {
		regSpec.RegistrationRequest = &v1.RegistrationRequest{
			RegistrationCodeSecretRef: params.regCodeSecretRef,
		}
	} else if params.regType == v1.RegistrationModeOffline && params.hasOfflineCertData {
		regSpec.OfflineRegistrationCertificateSecretRef = params.offlineCertSecretRef
	}

	// check if params has regURL and use, otherwise check if devmode and when true use staging Scc url
	if params.regURL != "" {
		regSpec.RegistrationRequest.RegistrationAPIUrl = &params.regURL
	}

	return regSpec
}

func (h *handler) regCodeFromSecretEntrypoint(params RegistrationParams) (*corev1.Secret, error) {
	secretName := params.regCodeSecretRef.Name

	regcodeSecret, err := h.secretRepo.Cache.Get(h.options.SystemNamespace, secretName)
	if err != nil && apierrors.IsNotFound(err) {
		regcodeSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: h.options.SystemNamespace,
				Name:      secretName,
			},
			Data: map[string][]byte{
				consts.SecretKeyRegistrationCode: params.regCode,
			},
		}
	}

	if regcodeSecret.Labels == nil {
		regcodeSecret.Labels = map[string]string{}
	}
	defaultLabels := params.Labels()
	defaultLabels[consts.LabelSccSecretRole] = string(consts.RegistrationCode)
	maps.Copy(regcodeSecret.Labels, defaultLabels)

	if !shared.SecretHasRegCodeFinalizer(regcodeSecret) {
		regcodeSecret = shared.SecretAddRegCodeFinalizer(regcodeSecret)
	}

	return regcodeSecret, nil
}

func (h *handler) offlineCertFromSecretEntrypoint(params RegistrationParams) (*corev1.Secret, error) {
	secretName := consts.OfflineCertificateSecretName(params.nameID)

	offlineCertSecret, err := h.secretRepo.Cache.Get(h.options.SystemNamespace, secretName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			offlineCertSecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: h.options.SystemNamespace,
					Name:      secretName,
				},
				Data: map[string][]byte{
					consts.SecretKeyOfflineRegCert: *params.offlineCertData,
				},
			}
		}
	}

	if offlineCertSecret.Labels == nil {
		offlineCertSecret.Labels = map[string]string{}
	}
	defaultLabels := params.Labels()
	defaultLabels[consts.LabelSccSecretRole] = string(consts.OfflineCertificate)
	maps.Copy(offlineCertSecret.Labels, defaultLabels)

	if !shared.SecretHasOfflineFinalizer(offlineCertSecret) {
		offlineCertSecret = shared.SecretAddOfflineFinalizer(offlineCertSecret)
	}

	return offlineCertSecret, nil
}
