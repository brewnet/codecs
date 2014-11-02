package form

import "github.com/stretchr/codecs/services"

var defaultCodecService services.CodecService

func CodecService() services.CodecService {
	if defaultCodecService == nil {
		defaultCodecService = services.NewWebCodecService()
	}
	return defaultCodecService
}

func SetCodecService(service services.CodecService) {
	defaultCodecService = service
}
