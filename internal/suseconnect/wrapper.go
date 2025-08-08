package suseconnect

import (
	"fmt"

	"github.com/SUSE/connect-ng/pkg/connection"
	"github.com/SUSE/connect-ng/pkg/registration"
	"github.com/pkg/errors"
	"github.com/rancher/scc-operator/internal/telemetry"

	rootLog "github.com/rancher/scc-operator/internal/log"
	v1 "github.com/rancher/scc-operator/pkg/apis/scc.cattle.io/v1"
)

func sccContextLogger() rootLog.StructuredLogger {
	logBuilder := rootLog.NewStructuredLoggerBuilder("suse-connect")
	return logBuilder.ToLogger()
}

type SccWrapper struct {
	rancherURL     string
	credentials    connection.Credentials
	conn           *connection.ApiConnection
	registered     *bool // only used by online mode
	rancherMetrics telemetry.MetricsWrapper
}

func DefaultConnectionOptions(appName, version string) connection.Options {
	// So this doesn't necessarily mean these have to match Rancher on the cluster.
	// Rather the details about the HTTP client talking to SCC
	// TODO: eventually add localization support?
	return connection.DefaultOptions(appName, version, "en_US")
}

type OnlineConnectionParams struct {
	RancherURL      string
	RegistrationURL string
	Options         connection.Options
}

func OnlineRancherConnection(
	params OnlineConnectionParams,
	credentials connection.Credentials,
	rancherMetrics telemetry.MetricsWrapper,
) SccWrapper {
	if credentials == nil {
		panic("credentials must be set")
	}

	registered := false
	if credentials.HasAuthentication() {
		registered = true
	}

	if params.RegistrationURL != "" {
		params.Options.URL = params.RegistrationURL
	}

	return SccWrapper{
		// TODO should use setting ServerURL lookup
		rancherURL:     params.RancherURL,
		credentials:    credentials,
		conn:           connection.New(params.Options, credentials),
		registered:     &registered,
		rancherMetrics: rancherMetrics,
	}
}

func OfflineRancherRegistration(
	rancherURL string,
	rancherMetrics telemetry.MetricsWrapper,
) SccWrapper {
	return SccWrapper{
		rancherURL:     rancherURL,
		rancherMetrics: rancherMetrics,
	}
}

type RegistrationSystemID int

func (id RegistrationSystemID) Int() int {
	return int(id)
}

func (id RegistrationSystemID) Ptr() *int {
	i := int(id)
	return &i
}

// Define constant values for empty and error
const (
	EmptyRegistrationSystemID     RegistrationSystemID = 0  // Used if an error happened before registration
	ErrorRegistrationSystemID     RegistrationSystemID = -1 // Used when error is related to registration
	KeepAliveRegistrationSystemID RegistrationSystemID = -2 // Indicates the Registration was handled via keepalive instead
	OfflineRegistrationSystemID   RegistrationSystemID = -3
)

func (sw *SccWrapper) SystemRegistration(regCode string) (RegistrationSystemID, error) {
	id, regErr := registration.Register(sw.conn, regCode, sw.rancherURL, sw.rancherMetrics.ToSystemInformation(), registration.NoExtraData)
	if regErr != nil {
		return ErrorRegistrationSystemID, errors.Wrap(regErr, "Cannot register system to SCC")
	}

	return RegistrationSystemID(id), nil
}

func (sw *SccWrapper) PrepareOfflineRegistrationRequest() (*registration.OfflineRequest, error) {
	identifier, version, arch := sw.rancherMetrics.GetProductIdentifier()
	rancherUUID := sw.rancherMetrics.GetRancherUUID()
	return registration.BuildOfflineRequest(identifier, version, arch, rancherUUID, sw.rancherMetrics.ToSystemInformation()), nil
}

func (sw *SccWrapper) KeepAlive() error {
	// 1 call Status
	status, statusErr := registration.Status(sw.conn, sw.rancherURL, sw.rancherMetrics.ToSystemInformation(), registration.NoExtraData)
	if status != registration.Registered {
		return fmt.Errorf("trying to send keepalive on a system that is not yet registered. register this system first: %v", statusErr)
	}
	// 2 verify response says we're registered still
	return statusErr
}

func (sw *SccWrapper) RegisterOrKeepAlive(regCode string) (RegistrationSystemID, error) {
	if *sw.registered {
		return KeepAliveRegistrationSystemID, sw.KeepAlive()
	}

	return sw.SystemRegistration(regCode)
}

func (sw *SccWrapper) Activate(regCode string) (*registration.Metadata, *registration.Product, error) {
	identifier, version, arch := sw.rancherMetrics.GetProductIdentifier()
	metaData, product, err := registration.Activate(sw.conn, identifier, version, arch, regCode)
	if err != nil {
		return nil, nil, err
	}

	return metaData, product, err
}

func (sw *SccWrapper) ActivationStatus() ([]*registration.Activation, error) {
	activations, err := registration.FetchActivations(sw.conn)
	if err != nil {
		return nil, err
	}
	return activations, nil
}

func (sw *SccWrapper) ProductInfo() (*registration.Product, error) {
	identifier, version, arch := sw.rancherMetrics.GetProductIdentifier()
	return registration.FetchProductInfo(sw.conn, identifier, version, arch)
}

func (sw *SccWrapper) Deregister() error {
	return registration.Deregister(sw.conn)
}

func PrepareSccURL(regIn *v1.Registration) string {
	if regIn != nil && regIn.Spec.RegistrationRequest != nil && regIn.Spec.RegistrationRequest.RegistrationAPIUrl != nil {
		return *regIn.Spec.RegistrationRequest.RegistrationAPIUrl
	}
	return ""
}
