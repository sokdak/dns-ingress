package environment

import (
	"github.com/go-openapi/swag"
	"github.com/sokdak/dns-ingress/pkg/common"
	"os"
	"strconv"
)

var (
	CloudflareAuthKey             *string
	CloudflareAuthEmail           *string
	CloudflareClientDebugMode     *bool
	CloudflareClientRateLimit     *float64
	CloudflareClientRetryMaxDelay *int
	CloudflareClientRetryMinDelay *int
	CloudflareClientRetryMaxCount *int
)

func LoadEnvs() {
	CloudflareAuthKey = getStringEnvOrNil("CLOUDFLARE_AUTH_KEY")
	CloudflareAuthEmail = getStringEnvOrNil("CLOUDFLARE_AUTH_EMAIL")
	CloudflareClientDebugMode = getBoolEnvOrDefault("CLOUDFLARE_CLIENT_DEBUG_MODE", false)
	CloudflareClientRateLimit = getFloat64EnvOrDefault("CLOUDFLARE_CLIENT_RATE_LIMIT", 4.0)
	CloudflareClientRetryMaxDelay = getIntEnvOrDefault("CLOUDFLARE_CLIENT_RETRY_MAX_DELAY", 3)
	CloudflareClientRetryMinDelay = getIntEnvOrDefault("CLOUDFLARE_CLIENT_RETRY_MIN_DELAY", 1)
	CloudflareClientRetryMaxCount = getIntEnvOrDefault("CLOUDFLARE_CLIENT_RETRY_MAX_COUNT", 30)
}

func getEnvOrNil(key string) *string {
	env, ok := os.LookupEnv(key)
	if !ok {
		return nil
	}
	return common.StringPointer(env)
}

func getStringEnvOrNil(key string) *string {
	return getEnvOrNil(key)
}

func getBoolEnvOrNil(key string) *bool {
	env := getEnvOrNil(key)
	if env == nil {
		return nil
	}
	if *env == "true" || *env == "True" {
		return common.BoolPointer(true)
	}
	if *env == "false" || *env == "False" {
		return common.BoolPointer(false)
	}
	return nil
}

func getIntEnvOrNil(key string) *int {
	env := getEnvOrNil(key)
	if env == nil {
		return nil
	}
	i, err := strconv.Atoi(*env)
	if err != nil {
		return nil
	}
	return common.IntPointer(i)
}

func getFloat64EnvOrNil(key string) *float64 {
	env := getEnvOrNil(key)
	if env == nil {
		return nil
	}
	i, err := swag.ConvertFloat64(*env)
	if err != nil {
		return nil
	}
	return common.Float64Pointer(i)
}

func getStringEnvOrDefault(key string, def string) *string {
	e := getStringEnvOrNil(key)
	if e != nil {
		return e
	}
	return common.StringPointer(def)
}

func getBoolEnvOrDefault(key string, def bool) *bool {
	e := getBoolEnvOrNil(key)
	if e != nil {
		return e
	}
	return common.BoolPointer(def)
}

func getIntEnvOrDefault(key string, def int) *int {
	e := getIntEnvOrNil(key)
	if e != nil {
		return e
	}
	return common.IntPointer(def)
}

func getFloat64EnvOrDefault(key string, def float64) *float64 {
	e := getFloat64EnvOrNil(key)
	if e != nil {
		return e
	}
	return common.Float64Pointer(def)
}
