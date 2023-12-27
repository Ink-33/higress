// Copyright (c) 2023 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handler

import (
	"net/url"
	"strings"

	cfg "github.com/alibaba/higress/plugins/wasm-go/extensions/jwt-auth/config"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
)

// extracToken 从三个来源中依次尝试抽取Token，若找不到Token则返回空字符串
func extractToken(keepToken bool, consumer *cfg.Consumer, log wrapper.Log) string {
	token := ""

	// 1. 从header中抽取token
	if h := consumer.FromHeaders; h != nil {
		token = extractFromHeader(keepToken, *h, log)
	}
	if token != "" {
		return token
	}

	// 2. 从params中抽取token
	if p := consumer.FromParams; p != nil {
		token = extractFromParams(keepToken, *p, log)
	}
	if token != "" {
		return token
	}

	// 3. 从cookies中抽取token
	if c := consumer.FromCookies; c != nil {
		token = extractFromCookies(keepToken, *c, log)
	}

	// 此处无需判空
	return token
}

func extractFromHeader(keepToken bool, headers []cfg.FromHeader, log wrapper.Log) (token string) {
	for i := range headers {
		token, err := proxywasm.GetHttpRequestHeader(headers[i].Name)
		if err != nil {
			log.Warnf("failed to get authorization: %v", err)
			continue
		}

		if token != "" {
			if !strings.HasPrefix(token, headers[i].ValuePrefix) {
				log.Warnf("authorization has no prefix %q", headers[i].ValuePrefix)
				return ""
			}
			if !keepToken {
				_ = proxywasm.RemoveHttpRequestHeader(headers[i].Name)
			}
			return strings.TrimPrefix(token, headers[i].ValuePrefix)
		}
	}
	return ""
}

func extractFromParams(keepToken bool, params []string, log wrapper.Log) (token string) {
	urlparams, err := proxywasm.GetHttpRequestHeader(":path")
	if err != nil {
		log.Warnf("failed to get authorization: %v", err)
		return ""
	}

	url, _ := url.Parse(urlparams)
	query := url.Query()

	for i := range params {
		token := query.Get(params[i])
		if token != "" {
			if !keepToken {
				query.Del(params[i])
			}
			return token
		}
	}
	return ""
}

func extractFromCookies(keepToken bool, cookies []string, log wrapper.Log) (token string) {
	requestCookies, err := proxywasm.GetHttpRequestHeader("Cookie")
	if err != nil {
		log.Warnf("failed to get authorization: %v", err)
		return ""
	}

	for i := range cookies {
		token := findCookie(requestCookies, cookies[i])
		if token != "" {
			if !keepToken {
				_ = proxywasm.ReplaceHttpRequestHeader("Cookie", deleteCookie(requestCookies, cookies[i]))
			}
			return token
		}
	}

	return ""
}

func findCookie(cookie string, key string) string {
	value := ""
	pairs := strings.Split(cookie, ";")

	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		kv := strings.Split(pair, "=")
		if kv[0] == key {
			value = kv[1]
			break
		}
	}
	return value
}

func deleteCookie(cookie string, key string) string {
	result := ""
	pairs := strings.Split(cookie, ";")

	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if !strings.HasPrefix(pair, key) {
			result += pair + ";"
		}
	}
	return strings.TrimSuffix(result, ";")
}
