/*
Copyright © 2021 Yugabyte Support

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package yugaware

import (
	"github.com/yugabyte/yb-tools/pkg/util"
	"go.uber.org/zap/zapcore"
)

type AuthToken struct {
	AuthToken    string `json:"authToken"`
	CustomerUUID string `json:"customerUUID"`
	UserUUID     string `json:"userUUID"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *LoginRequest) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("email", r.Email)
	enc.AddString("password", util.MaskOut(r.Password))
	return nil
}

type LoginResponse AuthToken

type RegisterYugawareRequest struct {
	Code            string `json:"code"`
	Name            string `json:"name"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirmPassword"`
	ConfirmEula     bool   `json:"confirmEula"`
}

type RegisterYugawareResponse AuthToken
