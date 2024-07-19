package spack

import "time"

type options struct {
	remote                string
	remoteUpdateFrequency time.Duration
	cacheDir              string
}

type Option func(*options)

func Remote(url string, updateFrequency time.Duration) Option {
	return func(o *options) {
		o.remote = url
		o.remoteUpdateFrequency = updateFrequency
	}
}

func CacheDir(path string) Option {
	return func(o *options) {
		o.cacheDir = path
	}
}
