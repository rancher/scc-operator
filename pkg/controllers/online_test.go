package controllers

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/SUSE/connect-ng/pkg/connection"
	sccreg "github.com/SUSE/connect-ng/pkg/registration"
	"github.com/rancher/scc-operator/internal/consts"
	"github.com/rancher/scc-operator/internal/logging"
	"github.com/rancher/scc-operator/internal/repos/secretrepo"
	v1 "github.com/rancher/scc-operator/pkg/apis/scc.cattle.io/v1"
	"github.com/rancher/wrangler/v3/pkg/generic/fake"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSubscriptionInfoNeedsRefresh(t *testing.T) {
	asserts := assert.New(t)

	hashMatches := "testhash123"
	hashMismatch := "differenthash456"

	tests := []struct {
		name        string
		subInfo     *v1.SubscriptionInfo
		regCodeHash string
		want        bool
	}{
		{
			name:        "nil subscription info returns true",
			subInfo:     nil,
			regCodeHash: hashMatches,
			want:        true,
		},
		{
			name: "hash mismatch returns true",
			subInfo: &v1.SubscriptionInfo{
				RegCodeHash: hashMatches,
			},
			regCodeHash: hashMismatch,
			want:        true,
		},
		{
			name: "has match returns false",
			subInfo: &v1.SubscriptionInfo{
				RegCodeHash: hashMatches,
			},
			regCodeHash: hashMatches,
			want:        false,
		},
		{
			name: "has match and non-expired returns false",
			subInfo: &v1.SubscriptionInfo{
				RegCodeHash: hashMatches,
				ExpiresAt:   &metav1.Time{Time: time.Now().Add(1 * time.Hour)},
			},
			regCodeHash: hashMatches,
			want:        false,
		},
		{
			name: "has match but expired returns true",
			subInfo: &v1.SubscriptionInfo{
				RegCodeHash: hashMatches,
				ExpiresAt:   &metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
			},
			regCodeHash: hashMatches,
			want:        true,
		},
		{
			name: "has match and zero expiresAt returns false",
			subInfo: &v1.SubscriptionInfo{
				RegCodeHash: hashMatches,
				ExpiresAt:   &metav1.Time{},
			},
			regCodeHash: hashMatches,
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := subscriptionInfoNeedsRefresh(tt.subInfo, tt.regCodeHash)
			asserts.Equal(tt.want, got)
		})
	}
}

func TestEnrichRegistrationError(t *testing.T) {
	asserts := assert.New(t)

	nonRecoverableErr := &connection.ApiError{Code: 422}
	recoverableErr := &connection.ApiError{Code: 429}
	networkErr := errors.New("connection timed out")

	tests := []struct {
		name    string
		regErr  error
		subInfo *v1.SubscriptionInfo
		wantErr string
	}{
		{
			name:    "returns original error when error is nil",
			regErr:  nil,
			subInfo: &v1.SubscriptionInfo{ProductClasses: []v1.ProductClass{{Name: "SLES"}}},
			wantErr: "",
		},
		{
			name:    "returns original error when subInfo is nil",
			regErr:  nonRecoverableErr,
			subInfo: nil,
			wantErr: nonRecoverableErr.Error(),
		},
		{
			name:   "returns original error when product classes list is empty",
			regErr: nonRecoverableErr,
			subInfo: &v1.SubscriptionInfo{
				ProductClasses: []v1.ProductClass{},
			},
			wantErr: nonRecoverableErr.Error(),
		},
		{
			name:   "returns original error when error is recoverable (e.g. 429)",
			regErr: recoverableErr,
			subInfo: &v1.SubscriptionInfo{
				ProductClasses: []v1.ProductClass{{Name: "SLES"}},
			},
			wantErr: recoverableErr.Error(),
		},
		{
			name:   "returns original error when error is generic network error",
			regErr: networkErr,
			subInfo: &v1.SubscriptionInfo{
				ProductClasses: []v1.ProductClass{{Name: "SLES"}},
			},
			wantErr: "connection timed out",
		},
		{
			name:   "formats error correctly with product descriptions for non-recoverable error",
			regErr: nonRecoverableErr,
			subInfo: &v1.SubscriptionInfo{
				ProductClasses: []v1.ProductClass{
					{Name: "SLES", Description: "SUSE Linux Enterprise Server 15 SP5"},
				},
			},
			wantErr: "the reg code provided is for SUSE Linux Enterprise Server 15 SP5 (original error: " + nonRecoverableErr.Error() + ")",
		},
		{
			name:   "falls back to product name if description is empty",
			regErr: nonRecoverableErr,
			subInfo: &v1.SubscriptionInfo{
				ProductClasses: []v1.ProductClass{
					{Name: "SLES-15-SP5", Description: ""},
				},
			},
			wantErr: "the reg code provided is for SLES-15-SP5 (original error: " + nonRecoverableErr.Error() + ")",
		},
		{
			name:   "supports multiple product classes",
			regErr: nonRecoverableErr,
			subInfo: &v1.SubscriptionInfo{
				ProductClasses: []v1.ProductClass{
					{Name: "SLES", Description: "SLES 15 SP5"},
					{Name: "SUMA", Description: "SUSE Manager"},
				},
			},
			wantErr: "the reg code provided is for SLES 15 SP5, SUSE Manager (original error: " + nonRecoverableErr.Error() + ")",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := enrichRegistrationError(tt.regErr, tt.subInfo)
			if tt.wantErr == "" {
				asserts.Nil(got)
			} else {
				asserts.NotNil(got)
				asserts.Equal(tt.wantErr, got.Error())
			}
		})
	}
}

func TestGetSubscriptionInfoFromSecret(t *testing.T) {
	asserts := assert.New(t)

	tests := []struct {
		name      string
		regSecret *corev1.Secret
		wantName  string
		wantErr   bool
		wantNil   bool
	}{
		{
			name:      "nil secret returns error",
			regSecret: nil,
			wantName:  "",
			wantErr:   true,
			wantNil:   true,
		},
		{
			name: "secret with nil annotations returns error",
			regSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: nil,
				},
			},
			wantName: "",
			wantErr:  true,
			wantNil:  true,
		},
		{
			name: "secret missing annotation returns error",
			regSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"other-annotation": "hello",
					},
				},
			},
			wantName: "",
			wantErr:  true,
			wantNil:  true,
		},
		{
			name: "secret with invalid JSON returns error",
			regSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"scc.cattle.io/subscription-info": "invalid-json",
					},
				},
			},
			wantName: "",
			wantErr:  true,
			wantNil:  true,
		},
		{
			name: "secret with null JSON annotation returns nil without error",
			regSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"scc.cattle.io/subscription-info": "null",
					},
				},
			},
			wantName: "",
			wantErr:  false,
			wantNil:  true,
		},
		{
			name: "secret with valid JSON returns subscription info",
			regSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"scc.cattle.io/subscription-info": `{"kind":"subscription","name":"Test Subscription","productClasses":[{"name":"SLES","description":"SLES 15"}]}`,
					},
				},
			},
			wantName: "Test Subscription",
			wantErr:  false,
			wantNil:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getSubscriptionInfoFromSecret(tt.regSecret)
			if tt.wantErr {
				asserts.Error(err)
				asserts.Nil(got)
			} else {
				asserts.NoError(err)
				if tt.wantNil {
					asserts.Nil(got)
				} else {
					asserts.NotNil(got)
					asserts.Equal(tt.wantName, got.Name)
					asserts.Len(got.ProductClasses, 1)
					asserts.Equal("SLES 15", got.ProductClasses[0].Description)
				}
			}
		})
	}
}

func TestSubscriptionInfoClearing(t *testing.T) {
	asserts := assert.New(t)

	tests := []struct {
		name             string
		registrationCode string
		regCodeHash      string
		subInfo          *v1.SubscriptionInfo
		wantSubInfoNil   bool
	}{
		{
			name:             "cleared when registrationCode is empty",
			registrationCode: "",
			regCodeHash:      "",
			subInfo: &v1.SubscriptionInfo{
				RegCodeHash: "somehash",
			},
			wantSubInfoNil: true,
		},
		{
			name:             "cleared on hash mismatch when refresh is needed",
			registrationCode: "newcode",
			regCodeHash:      "newhash",
			subInfo: &v1.SubscriptionInfo{
				RegCodeHash: "oldhash",
			},
			wantSubInfoNil: true,
		},
		{
			name:             "not cleared on hash match (no refresh needed)",
			registrationCode: "samecode",
			regCodeHash:      "samehash",
			subInfo: &v1.SubscriptionInfo{
				RegCodeHash: "samehash",
			},
			wantSubInfoNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registrationObj := &v1.Registration{
				Status: v1.RegistrationStatus{
					SubscriptionInfo: tt.subInfo,
				},
			}

			if tt.registrationCode == "" {
				registrationObj.Status.SubscriptionInfo = nil
			} else {
				if subscriptionInfoNeedsRefresh(registrationObj.Status.SubscriptionInfo, tt.regCodeHash) {
					if registrationObj.Status.SubscriptionInfo != nil && registrationObj.Status.SubscriptionInfo.RegCodeHash != tt.regCodeHash {
						registrationObj.Status.SubscriptionInfo = nil
					}
				}
			}

			if tt.wantSubInfoNil {
				asserts.Nil(registrationObj.Status.SubscriptionInfo)
			} else {
				asserts.NotNil(registrationObj.Status.SubscriptionInfo)
				asserts.Equal(tt.regCodeHash, registrationObj.Status.SubscriptionInfo.RegCodeHash)
			}
		})
	}
}

func TestMapSubscriptionInfo(t *testing.T) {
	asserts := assert.New(t)

	now := time.Now()
	regCodeHash := "hash123"

	// Case 1: Nil input
	res, covered := mapSubscriptionInfo(nil, regCodeHash)
	asserts.Nil(res)
	asserts.Nil(covered)

	// Case 2: Standard input (< 50 product classes)
	subInfoSmall := &sccreg.SubscriptionInfo{
		Kind:          "subscription",
		Name:          "Small Sub",
		StartsAt:      now,
		ExpiresAt:     now.Add(time.Hour),
		Limit:         5,
		Notifications: "alert",
		ProductClasses: []sccreg.ProductClass{
			{Name: "SLES", Description: "SUSE Linux Enterprise Server"},
			{Name: "SUMA", Description: ""},
		},
	}

	res, covered = mapSubscriptionInfo(subInfoSmall, regCodeHash)
	asserts.NotNil(res)
	asserts.Equal("subscription", res.Kind)
	asserts.Equal("Small Sub", res.Name)
	asserts.Equal(now.Unix(), res.StartsAt.Unix())
	asserts.Equal(now.Add(time.Hour).Unix(), res.ExpiresAt.Unix())
	asserts.Equal(5, res.Limit)
	asserts.Equal("alert", res.Notifications)
	asserts.Equal(regCodeHash, res.RegCodeHash)

	asserts.Len(res.ProductClasses, 2)
	asserts.Equal("SLES", res.ProductClasses[0].Name)
	asserts.Equal("SUSE Linux Enterprise Server", res.ProductClasses[0].Description)
	asserts.Equal("SUMA", res.ProductClasses[1].Name)
	asserts.Equal("", res.ProductClasses[1].Description)

	asserts.Len(covered, 2)
	asserts.Equal("SUSE Linux Enterprise Server", covered[0])
	asserts.Equal("SUMA", covered[1])

	// Case 3: Overflow input (> 50 product classes)
	largeProductClasses := []sccreg.ProductClass{}
	for i := 1; i <= 55; i++ {
		largeProductClasses = append(largeProductClasses, sccreg.ProductClass{
			Name:        fmt.Sprintf("PROD-%d", i),
			Description: fmt.Sprintf("Desc-%d", i),
		})
	}
	subInfoLarge := &sccreg.SubscriptionInfo{
		Kind:           "subscription",
		Name:           "Large Sub",
		ProductClasses: largeProductClasses,
	}

	res, covered = mapSubscriptionInfo(subInfoLarge, regCodeHash)
	asserts.NotNil(res)
	// pcs should be truncated (length 50)
	asserts.Len(res.ProductClasses, 50)
	// coveredProductNames should still be fully populated (all 55 elements)
	asserts.Len(covered, 55)
	asserts.Equal("Desc-1", covered[0])
	asserts.Equal("Desc-55", covered[54])

	// Case 4: Zero times
	subInfoZeroTime := &sccreg.SubscriptionInfo{
		StartsAt:  time.Time{},
		ExpiresAt: time.Time{},
	}
	res, _ = mapSubscriptionInfo(subInfoZeroTime, regCodeHash)
	asserts.NotNil(res)
	asserts.Nil(res.StartsAt)
	asserts.Nil(res.ExpiresAt)
}

func TestRestoreSubscriptionInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockController := fake.NewMockControllerInterface[*corev1.Secret, *corev1.SecretList](ctrl)
	mockCache := fake.NewMockCacheInterface[*corev1.Secret](ctrl)
	repo := &secretrepo.SecretRepository{Controller: mockController, Cache: mockCache}

	sccOnline := &sccOnlineMode{
		secretRepo: repo,
		log:        logging.NewLog(),
	}

	// Case 1: SubscriptionInfo already present
	reg1 := &v1.Registration{
		Status: v1.RegistrationStatus{
			SubscriptionInfo: &v1.SubscriptionInfo{Name: "Existing"},
		},
	}
	sccOnline.restoreSubscriptionInfo(reg1)
	assert.Equal(t, "Existing", reg1.Status.SubscriptionInfo.Name)

	// Case 2: Spec.RegistrationRequest or SecretRef is nil
	reg2 := &v1.Registration{}
	sccOnline.restoreSubscriptionInfo(reg2)
	assert.Nil(t, reg2.Status.SubscriptionInfo)

	// Case 3: Secret exists and contains subscription info annotation
	reg3 := &v1.Registration{
		Spec: v1.RegistrationSpec{
			RegistrationRequest: &v1.RegistrationRequest{
				RegistrationCodeSecretRef: &corev1.SecretReference{
					Namespace: "ns",
					Name:      "secret-name",
				},
			},
		},
	}
	secretWithAnnotation := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "secret-name",
			Annotations: map[string]string{
				consts.AnnotationSubscriptionInfo: `{"kind":"subscription","name":"My Sub"}`,
			},
		},
	}
	mockCache.EXPECT().Get("ns", "secret-name").Return(secretWithAnnotation, nil).Times(1)

	sccOnline.restoreSubscriptionInfo(reg3)
	assert.NotNil(t, reg3.Status.SubscriptionInfo)
	assert.Equal(t, "My Sub", reg3.Status.SubscriptionInfo.Name)
}

func TestUpdateRegistrationSecret(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockController := fake.NewMockControllerInterface[*corev1.Secret, *corev1.SecretList](ctrl)
	mockCache := fake.NewMockCacheInterface[*corev1.Secret](ctrl)
	repo := &secretrepo.SecretRepository{Controller: mockController, Cache: mockCache}

	sccOnline := &sccOnlineMode{
		secretRepo: repo,
		log:        logging.NewLog(),
	}

	// Case 1: No secret reference
	reg1 := &v1.Registration{}
	sccOnline.updateRegistrationSecret(reg1, nil) // Should do nothing and not fail

	// Case 2: Secret has no changes
	reg2 := &v1.Registration{
		Spec: v1.RegistrationSpec{
			RegistrationRequest: &v1.RegistrationRequest{
				RegistrationCodeSecretRef: &corev1.SecretReference{
					Namespace: "ns",
					Name:      "secret-name",
				},
			},
		},
		Status: v1.RegistrationStatus{
			SubscriptionInfo: &v1.SubscriptionInfo{
				Name: "Sub",
			},
		},
	}
	existingSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "secret-name",
			Annotations: map[string]string{
				consts.AnnotationSubscriptionInfo: `{"name":"Sub"}`,
			},
		},
		Data: map[string][]byte{
			consts.SecretKeyCoveredProducts: []byte("ProdA, ProdB"),
		},
	}
	// Called once by secretRepo.Get (CreateOrUpdateSecret is not called since changed is false)
	mockCache.EXPECT().Get("ns", "secret-name").Return(existingSecret, nil).Times(1)

	// Since annotations and data match, no update should be performed
	sccOnline.updateRegistrationSecret(reg2, []string{"ProdA", "ProdB"})

	// Case 3: Secret has changes and is updated
	reg3 := &v1.Registration{
		Spec: v1.RegistrationSpec{
			RegistrationRequest: &v1.RegistrationRequest{
				RegistrationCodeSecretRef: &corev1.SecretReference{
					Namespace: "ns",
					Name:      "secret-name",
				},
			},
		},
		Status: v1.RegistrationStatus{
			SubscriptionInfo: &v1.SubscriptionInfo{
				Name: "New Sub Name",
			},
		},
	}
	// Called once by secretRepo.Get, and again by secretRepo.CreateOrUpdateSecret (since changed is true)
	mockCache.EXPECT().Get("ns", "secret-name").Return(existingSecret, nil).Times(2)
	// We expect Patch to be called on controller during update
	mockController.EXPECT().Patch("ns", "secret-name", gomock.Any(), gomock.Any()).Return(existingSecret, nil).Times(1)

	sccOnline.updateRegistrationSecret(reg3, []string{"ProdNew"})
}
