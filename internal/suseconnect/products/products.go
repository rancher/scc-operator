package products

import "strings"

type SCCProduct int

const (
	ProductNone SCCProduct = iota
	ProductRancher
)

var (
	productNames = map[SCCProduct]string{
		ProductNone:    "",
		ProductRancher: "rancher",
	}
	productLookup = map[string]SCCProduct{}
)

func init() {
	for product, productName := range productNames {
		productLookup[productName] = product
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
	name, ok := productNames[s]
	if !ok {
		return ProductName("unknown")
	}

	return ProductName(name)
}

func (s SCCProduct) String() string {
	return productNames[s]
}
