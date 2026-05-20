/*
 * Copyright 2025 superagent-ai Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
)

// Sign returns an HMAC-SHA256 signature of the form "sha256=<hex>".
// The signed payload is "<timestamp>.<body>" which binds the timestamp to the
// body, preventing replay attacks when receivers enforce a timestamp window.
func Sign(secret string, timestamp int64, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%d.", timestamp)))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// Verify checks whether signature matches Sign(secret, timestamp, body).
// The comparison is done in constant time to mitigate timing side-channels.
func Verify(secret string, timestamp int64, body []byte, signature string) bool {
	expected := Sign(secret, timestamp, body)
	return subtle.ConstantTimeCompare([]byte(expected), []byte(signature)) == 1
}
