package vault

type KubernetesAuthOpt func(k *kubernetesAuth) error

func WithMountPoint(mountPoint string) KubernetesAuthOpt {
	return func(k *kubernetesAuth) error {
		k.mountPoint = mountPoint

		return nil
	}
}

func WithJwt(jwt string) KubernetesAuthOpt {
	return func(k *kubernetesAuth) error {
		k.jwt = jwt

		return nil
	}
}

func WithJwtFromFile(path string) KubernetesAuthOpt {
	return func(k *kubernetesAuth) error {
		jwt, err := loadJwt(path)
		if err != nil {
			return err
		}

		k.jwt = jwt

		return nil
	}
}
