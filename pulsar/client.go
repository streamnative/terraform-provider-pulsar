package pulsar

// As an instance of pulsarctl can only handle one version at a time, this makes sure we can
// request the right version for the right API resource.

import (
	"errors"
	"github.com/streamnative/pulsarctl/pkg/pulsar"
	"github.com/streamnative/pulsarctl/pkg/pulsar/common"
	"strconv"
)

func getClientFromMeta(version common.APIVersion, meta interface{}) pulsar.Client {
	versions := meta.(map[common.APIVersion]pulsar.Client)
	return versions[version]
}

func getClientV1FromMeta(meta interface{}) pulsar.Client {
	return getClientFromMeta(common.V1, meta)
}

func getClientV2FromMeta(meta interface{}) pulsar.Client {
	return getClientFromMeta(common.V2, meta)
}

func getClientV3FromMeta(meta interface{}) pulsar.Client {
	return getClientFromMeta(common.V3, meta)
}

func fixClientIntConversion(fn func() (int, error)) (int, error) {
	rv, err := fn()

	if err != nil {
		if err, ok := err.(*strconv.NumError); ok && errors.Is(err, strconv.ErrSyntax) {
			// At least on 2.8 the default number on the api is blank and pulsarctl
			// will break because it tries to convert "" to an integer
			return 0, nil
		}
		return 0, err
	}
	return rv, nil
}
