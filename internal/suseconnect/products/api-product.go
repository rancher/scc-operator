package products

import "strings"

type OperatorProduct struct {
	product    SCCProduct
	Identifier string `json:"identifier"`
	Version    string `json:"version"`
	Arch       string `json:"arch"`
}

func NewOperatorProduct(identifier string, version string, arch string) *OperatorProduct {
	product := ParseProductName(identifier)
	if !product.IsValid() {
		logger.Warnf("Invalid product name: %s; will default to `unknown`", identifier)
	}
	return &OperatorProduct{
		product:    product,
		Identifier: product.String(),
		Version:    version,
		Arch:       arch,
	}
}

func (op OperatorProduct) sccSafeVersion() string {
	safeVersion, _ := strings.CutPrefix(op.Version, "v")
	return safeVersion
}

func (op OperatorProduct) ToTriplet() string {
	return op.Identifier + "/" + op.sccSafeVersion() + "/" + op.Arch
}

func (op OperatorProduct) GetTripletValues() (string, string, string) {
	return op.Identifier, op.sccSafeVersion(), op.Arch
}
