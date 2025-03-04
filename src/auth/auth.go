/*
 * Tencent is pleased to support the open source community by making 蓝鲸 available.
 * Copyright (C) 2017-2018 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package auth

import (
	"context"
	"errors"

	"configcenter/src/apimachinery/util"
	"configcenter/src/auth/authcenter"
	"configcenter/src/auth/meta"
)

type Authorize interface {
	Authorizer
	ResourceHandler
}

var NoAuthorizeError = errors.New("no authorize")

type Authorizer interface {
	// Authorize works to check if a user has the authority to operate resources.
	Authorize(ctx context.Context, a *meta.AuthAttribute) (decision meta.Decision, err error)
	AuthorizeBatch(ctx context.Context, user meta.UserInfo, resources ...meta.ResourceAttribute) (decisions []meta.Decision, err error)
	GetAuthorizedBusinessList(ctx context.Context, user meta.UserInfo) ([]int64, error)
	AdminEntrance(ctx context.Context, user meta.UserInfo) ([]string, error)
	GetAuthorizedAuditList(ctx context.Context, user meta.UserInfo, businessID int64) ([]authcenter.AuthorizedResource, error)
	Enabled() bool
}

// ResourceHandler is used to handle the resources register to authorize center.
// request id is a identifier for a request, returned by IAM.
type ResourceHandler interface {
	// register a resource
	RegisterResource(ctx context.Context, rs ...meta.ResourceAttribute) error
	// register a resource
	DryRunRegisterResource(ctx context.Context, rs ...meta.ResourceAttribute) (*authcenter.RegisterInfo, error)
	// deregister a resource
	DeregisterResource(ctx context.Context, rs ...meta.ResourceAttribute) error
	// deregister a resource with raw iam resource id
	RawDeregisterResource(ctx context.Context, scope authcenter.ScopeInfo, rs ...meta.BackendResource) error
	// update a resource's info
	UpdateResource(ctx context.Context, rs *meta.ResourceAttribute) error
	// get a resource's info
	Get(ctx context.Context) error
	// list resources by condition
	ListResources(ctx context.Context, r *meta.ResourceAttribute) ([]meta.BackendResource, error)
	// init the authcenter
	Init(ctx context.Context, config meta.InitConfig) error
}

// NewAuthorize is used to initialized a Authorize instance interface,
// which is used for request authorize and resource handle.
// This allows bk-cmdb to support other kind of auth center.
// tls can be nil if it is not care.
// authConfig is a way to parse configuration info for the connection to a auth center.
func NewAuthorize(tls *util.TLSClientConfig, authConfig authcenter.AuthConfig) (Authorize, error) {
	return authcenter.NewAuthCenter(tls, authConfig)
}
