package products

import (
	"strings"

	"github.com/rancher/scc-operator/pkg/util/log"
)

// This package implements helpers for SCC Operator supported products

var logger = log.NewComponentLogger("suseconnect.products")

type SCCProduct int

const (
	ProductNone SCCProduct = iota
	ProductRancher
)

var (
	productNamesPairs = map[SCCProduct][]string{
		ProductNone:    []string{"unknown"},
		ProductRancher: []string{"rancher", "rancher-prime"},
	}
	productLookup = map[string]SCCProduct{}
)

func init() {
	for product, productNames := range productNamesPairs {
		for _, productName := range productNames {
			productLookup[productName] = product
		}
	}
}

func ParseProductName(productName string) SCCProduct {
	if product, ok := productLookup[strings.ToLower(productName)]; ok {
		return product
	}

	return ProductNone
}

type ProductName string

func (s SCCProduct) ProductName() ProductName {
	name, ok := productNamesPairs[s]
	if !ok {
		return ProductName("unknown")
	}

	return ProductName(name[0])
}

func (s SCCProduct) String() string {
	return string(s.ProductName())
}
func (s SCCProduct) IsValid() bool {
	_, ok := productNamesPairs[s]
	return ok
}
