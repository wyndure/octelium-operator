package v1

func Resources() []Resource {
	return []Resource{&OcteliumClusterConfig{}, &OcteliumConfig{}, &OcteliumPolicy{}, &OcteliumIdentityProvider{}, &OcteliumNamespace{}, &OcteliumGroup{}, &OcteliumUser{}, &OcteliumService{}, &OcteliumCredential{}, &OcteliumSecret{}, &OcteliumRegion{}}
}
