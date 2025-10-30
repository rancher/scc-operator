package products

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ParseProductName(t *testing.T) {
	rancherProduct := ParseProductName("rancher")
	assert.Equal(t, ProductRancher, rancherProduct)
	assert.Equal(t, ProductName("rancher"), rancherProduct.ProductName())
	assert.Equal(t, "rancher", rancherProduct.String())

	harvesterProduct := ParseProductName("harvester")
	assert.Equal(t, ProductNone, harvesterProduct)
	assert.Equal(t, ProductName("unknown"), harvesterProduct.ProductName())
	assert.Equal(t, "unknown", harvesterProduct.String())
}

func Test_ParseProductName_Errors(t *testing.T) {
	fakeProduct := SCCProduct(4)
	assert.Equal(t, ProductName("unknown"), fakeProduct.ProductName())
	assert.Equal(t, "unknown", fakeProduct.String())
}

func Test_CustomProductName(t *testing.T) {
	harvesterProductCustom := ProductName("harvester")
	assert.Equal(t, ProductName("harvester"), harvesterProductCustom)

	harvesterProduct := ParseProductName("harvester")
	assert.NotEqual(t, harvesterProductCustom, harvesterProduct.ProductName())
}
