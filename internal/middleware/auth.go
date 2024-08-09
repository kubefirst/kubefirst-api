/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/env"
	"github.com/rs/zerolog/log"
)

// ValidateAPIKey determines whether or not a request is authenticated with a valid API key
func ValidateAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		APIKey := strings.TrimPrefix(c.Request.Header.Get("Authorization"), "Bearer ")

		if APIKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"status": 401, "message": "Authentication failed - no API key provided in request"})
			c.Abort()

			log.Info().Msg(" Request Status: 401;  Authentication failed - no API key provided in request")
			return
		}

		env, _ := env.GetEnv(constants.SilenceGetEnv)

		if APIKey != env.K1AccessToken {
			c.JSON(http.StatusUnauthorized, gin.H{"status": 401, "message": "Authentication failed - not a valid API key"})
			c.Abort()

			log.Info().Msg(" Request Status: 401;  Authentication failed - no API key provided in request")
			return
		}
	}
}
