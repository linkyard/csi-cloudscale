package driver

import (
	"context"
	"github.com/cloudscale-ch/cloudscale-go-sdk"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestCalculateStorageGBEmpty(t *testing.T) {
	value, err := calculateStorageGB(nil, "")
	assert.Equal(t, 1, value)
	assert.NoError(t, err)
}

func TestCalculateStorageGBLimitTooLow(t *testing.T) {
	_, err := calculateStorageGB(&csi.CapacityRange{LimitBytes: 1}, "")
	assert.Error(t, err)
}

func TestCalculateStorageGBNotPossible(t *testing.T) {
	base := int64(50 * GB)
	_, err := calculateStorageGB(&csi.CapacityRange{RequiredBytes: base + 1, LimitBytes: base + 2}, "")
	assert.Error(t, err)
}

func TestCalculateStorageGBEdges(t *testing.T) {
	base := int64(50 * GB)
	value, err := calculateStorageGB(&csi.CapacityRange{RequiredBytes: base, LimitBytes: base * 2}, "")
	assert.NoError(t, err)
	assert.Equal(t, 50, value)
}

func TestCalculateStorageGBRounding(t *testing.T) {
	base := int64(30 * GB)
	value, err := calculateStorageGB(&csi.CapacityRange{RequiredBytes: base + 1}, "")
	assert.NoError(t, err)
	assert.Equal(t, 31, value)

	value, err = calculateStorageGB(&csi.CapacityRange{RequiredBytes: base - 1}, "")
	assert.NoError(t, err)
	assert.Equal(t, 30, value)
}

func TestRequiredBulkStorageSize(t *testing.T) {
	// should default to the minimum storage size
	val, err := calcStorageGbBulk(200, 0)
	assert.NoError(t, err)
	assert.Equal(t, 200, val)
}
func TestDefaultToMinimumBulkStorageSize(t *testing.T) {
	// should default to the minimum storage size
	val, err := calcStorageGbBulk(0, 0)
	assert.NoError(t, err)
	assert.Equal(t, 100, val)
}
func TestLimitMustNotBeSmallerThanMinimumBulkStorageSize(t *testing.T) {
	// limit is smaller than minimum bulk disk size
	_, err := calcStorageGbBulk(5, 5)
	assert.Error(t, err)
}
func TestRequestedBytesSmallerThanMinimumSizeUsesMinimumBulkStorageSize(t *testing.T) {
	// use minimum bulk disk size if the limit allows it
	val, err := calcStorageGbBulk(5, 100)
	assert.NoError(t, err)
	assert.Equal(t, 100, val)
}

func calcStorageGbBulk(reqGb int, limitGb int) (int, error) {
	if reqGb == -1 {
		if limitGb == -1 {
			return calculateStorageGB(&csi.CapacityRange{
			}, "bulk")
		} else {
			return calculateStorageGB(&csi.CapacityRange{
				LimitBytes: int64(limitGb * GB),
			}, "bulk")
		}
	} else {
		if limitGb == -1 {
			return calculateStorageGB(&csi.CapacityRange{
				RequiredBytes: int64(reqGb * GB),
			}, "bulk")
		} else {
			return calculateStorageGB(&csi.CapacityRange{
				RequiredBytes: int64(reqGb * GB),
				LimitBytes:    int64(limitGb * GB),
			}, "bulk")
		}
	}
}

func TestCreateVolumeTypeSsdWithoutExplicitlySpecifyingTheType(t *testing.T) {
	driver, server := createDriverForTest(t)
	defer server.Close()

	volumeName := randString(32)

	response, err := driver.CreateVolume(
		context.Background(),
		makeCreateVolumeRequest(volumeName, 1, ""),
	)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotNil(t, response.Volume)
	assert.Equal(t, int64(1) * GB, response.Volume.CapacityBytes)
	assert.Equal(t, volumeName, response.Volume.VolumeContext[PublishInfoVolumeName])

	volumes, err := driver.cloudscaleClient.Volumes.List(context.Background(),&cloudscale.ListVolumeParams{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(volumes))
	assert.Equal(t, 1, volumes[0].SizeGB)
	assert.Equal(t, "ssd", volumes[0].Type)
}

func TestCreateVolumeTypeSsdExplicitlySpecifyingTheType(t *testing.T) {
	driver, server := createDriverForTest(t)
	defer server.Close()

	volumeName := randString(32)

	response, err := driver.CreateVolume(
		context.Background(),
		makeCreateVolumeRequest(volumeName, 5, "ssd"),
	)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotNil(t, response.Volume)
	assert.Equal(t, int64(5) * GB, response.Volume.CapacityBytes)
	assert.Equal(t, volumeName, response.Volume.VolumeContext[PublishInfoVolumeName])

	volumes, err := driver.cloudscaleClient.Volumes.List(context.Background(),&cloudscale.ListVolumeParams{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(volumes))
	assert.Equal(t, 5, volumes[0].SizeGB)
	assert.Equal(t, "ssd", volumes[0].Type)
}

func TestCreateVolumeTypeBulk(t *testing.T) {
	driver, server := createDriverForTest(t)
	defer server.Close()

	volumeName := randString(32)

	response, err := driver.CreateVolume(
		context.Background(),
		makeCreateVolumeRequest(volumeName, 100, "bulk"),
	)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotNil(t, response.Volume)
	assert.Equal(t, int64(100) * GB, response.Volume.CapacityBytes)
	assert.Equal(t, volumeName, response.Volume.VolumeContext[PublishInfoVolumeName])

	volumes, err := driver.cloudscaleClient.Volumes.List(context.Background(),&cloudscale.ListVolumeParams{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(volumes))
	assert.Equal(t, 100, volumes[0].SizeGB)
	assert.Equal(t, "bulk", volumes[0].Type)
}

func TestCreateVolumeInvalidType(t *testing.T) {
	driver, server := createDriverForTest(t)
	defer server.Close()

	volumeName := randString(32)

	_, err := driver.CreateVolume(
		context.Background(),
		makeCreateVolumeRequest(volumeName, 100, "foo"),
	)

	assert.Error(t, err)
}

func TestLuksEncryptionAttributeIsSetInContext(t *testing.T) {
	driver, server := createDriverForTest(t)
	defer server.Close()

	// explicitly set luks encryption to false
	volumeName := randString(32)
	response, err := driver.CreateVolume(
		context.Background(),
		makeLuksCreateVolumeRequest(volumeName, 100, "bulk", false),
	)
	assert.NoError(t, err)
	assert.Equal(t, "false", response.Volume.VolumeContext[LuksEncryptedAttribute])

	// explicitly set luks encryption to true
	volumeName = randString(32)
	response, err = driver.CreateVolume(
		context.Background(),
		makeLuksCreateVolumeRequest(volumeName, 100, "bulk", true),
	)
	assert.NoError(t, err)
	assert.Equal(t, "true", response.Volume.VolumeContext[LuksEncryptedAttribute])

	// don't set the luks encryption parameter - must implicitly default to false
	volumeName = randString(32)
	response, err = driver.CreateVolume(
		context.Background(),
		makeCreateVolumeRequest(volumeName, 100, "bulk"),
	)
	assert.NoError(t, err)
	assert.Equal(t, "false", response.Volume.VolumeContext[LuksEncryptedAttribute])
}

func makeLuksCreateVolumeRequest(volumeName string, sizeGb int, volumeType string, luksEncryptionEnabled bool) *csi.CreateVolumeRequest {
	request := makeCreateVolumeRequest(volumeName, sizeGb, volumeType)
	if luksEncryptionEnabled {
		request.Parameters["luksEncrypted"] = "true"
	} else {
		request.Parameters["luksEncrypted"] = "false"
	}
	return request
}

func makeCreateVolumeRequest(volumeName string, sizeGb int, volumeType string) *csi.CreateVolumeRequest {
	return &csi.CreateVolumeRequest{
		Name: volumeName,
		VolumeCapabilities: []*csi.VolumeCapability{
			{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
				AccessMode: &csi.VolumeCapability_AccessMode{
					Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
				},
			},
		},
		CapacityRange: &csi.CapacityRange{
			RequiredBytes: int64(sizeGb) * GB,
		},
		Parameters: map[string]string{
			"type": volumeType,
		},
	}
}

func createDriverForTest(t *testing.T) (*Driver, *httptest.Server) {
	serverId := "987654"
	fake := &fakeAPI{
		t:       t,
		volumes: map[string]*cloudscale.Volume{},
		servers: map[string]*cloudscale.Server{
			serverId: {},
		},
	}

	server := httptest.NewServer(fake)

	cloudscaleClient := cloudscale.NewClient(nil)
	serverUrl, _ := url.Parse(server.URL)
	cloudscaleClient.BaseURL = serverUrl

	return &Driver{
		mounter:          &fakeMounter{},
		log:              logrus.New().WithField("test_enabled", true),
		cloudscaleClient: cloudscaleClient,
	}, server
}
