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

package operation

import (
	"context"
	"sort"
	"strings"
	"sync"

	"configcenter/src/apimachinery"
	"configcenter/src/auth/extensions"
	authmeta "configcenter/src/auth/meta"
	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/condition"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/metadata"
	"configcenter/src/common/util"
	"configcenter/src/scene_server/topo_server/core/inst"
	"configcenter/src/scene_server/topo_server/core/model"
	"configcenter/src/scene_server/topo_server/core/types"
)

// BusinessOperationInterface business operation methods
type BusinessOperationInterface interface {
	CreateBusiness(params types.ContextParams, obj model.Object, data mapstr.MapStr) (inst.Inst, error)
	DeleteBusiness(params types.ContextParams, obj model.Object, bizID int64) error
	FindBusiness(params types.ContextParams, obj model.Object, fields []string, cond condition.Condition) (count int, results []inst.Inst, err error)
	GetInternalModule(params types.ContextParams, obj model.Object, bizID int64) (count int, result *metadata.InnterAppTopo, err error)
	UpdateBusiness(params types.ContextParams, data mapstr.MapStr, obj model.Object, bizID int64) error

	SetProxy(set SetOperationInterface, module ModuleOperationInterface, inst InstOperationInterface, obj ObjectOperationInterface)
}

// NewBusinessOperation create a business instance
func NewBusinessOperation(client apimachinery.ClientSetInterface, authManager *extensions.AuthManager) BusinessOperationInterface {
	return &business{
		clientSet:   client,
		authManager: authManager,
	}
}

type business struct {
	clientSet   apimachinery.ClientSetInterface
	authManager *extensions.AuthManager
	inst        InstOperationInterface
	set         SetOperationInterface
	module      ModuleOperationInterface
	obj         ObjectOperationInterface
}

func (b *business) SetProxy(set SetOperationInterface, module ModuleOperationInterface, inst InstOperationInterface, obj ObjectOperationInterface) {
	b.inst = inst
	b.set = set
	b.module = module
	b.obj = obj
}
func (b *business) CreateBusiness(params types.ContextParams, obj model.Object, data mapstr.MapStr) (inst.Inst, error) {

	defaulFieldVal, err := data.Int64(common.BKDefaultField)
	if nil != err {
		blog.Errorf("[operation-biz] failed to create business, error info is did not set the default field, %s", err.Error())
		return nil, params.Err.New(common.CCErrTopoAppCreateFailed, err.Error())
	}
	if defaulFieldVal == int64(common.DefaultAppFlag) && params.SupplierAccount != common.BKDefaultOwnerID {
		// this is a new supplier owner and prepare to create a new business.
		asstQuery := map[string]interface{}{
			common.BKOwnerIDField: common.BKDefaultOwnerID,
		}
		defaultOwnerHeader := util.CloneHeader(params.Header)
		defaultOwnerHeader.Set(common.BKHTTPOwnerID, common.BKDefaultOwnerID)

		asstRsp, err := b.clientSet.CoreService().Association().ReadModelAssociation(context.Background(), defaultOwnerHeader, &metadata.QueryCondition{Condition: asstQuery})
		if nil != err {
			blog.Errorf("create business failed to get default assts, error info is %s", err.Error())
			return nil, params.Err.New(common.CCErrTopoAppCreateFailed, err.Error())
		}
		if !asstRsp.Result {
			return nil, params.Err.Error(asstRsp.Code)
		}
		expectAssts := asstRsp.Data.Info
		blog.Infof("copy asst for %s, %+v", params.SupplierAccount, expectAssts)

		existAsstRsp, err := b.clientSet.CoreService().Association().ReadModelAssociation(context.Background(), params.Header, &metadata.QueryCondition{Condition: asstQuery})
		if nil != err {
			blog.Errorf("create business failed to get default assts, error info is %s", err.Error())
			return nil, params.Err.New(common.CCErrTopoAppCreateFailed, err.Error())
		}
		if !existAsstRsp.Result {
			return nil, params.Err.Error(existAsstRsp.Code)
		}
		existAssts := existAsstRsp.Data.Info

	expectLoop:
		for _, asst := range expectAssts {
			asst.OwnerID = params.SupplierAccount
			for _, existAsst := range existAssts {
				if existAsst.ObjectID == asst.ObjectID &&
					existAsst.AsstObjID == asst.AsstObjID &&
					existAsst.AsstKindID == asst.AsstKindID {
					continue expectLoop
				}
			}

			var createAsstRsp *metadata.CreatedOneOptionResult
			var err error
			if asst.AsstKindID == common.AssociationKindMainline {
				// bk_mainline is a inner association type that can only create in special case,
				// so we separate bk_mainline association type creation with a independent method,
				createAsstRsp, err = b.clientSet.CoreService().Association().CreateMainlineModelAssociation(context.Background(), params.Header, &metadata.CreateModelAssociation{Spec: asst})
			} else {
				createAsstRsp, err = b.clientSet.CoreService().Association().CreateModelAssociation(context.Background(), params.Header, &metadata.CreateModelAssociation{Spec: asst})
			}
			if nil != err {
				blog.Errorf("create business failed to copy default assts, error info is %s", err.Error())
				return nil, params.Err.New(common.CCErrTopoAppCreateFailed, err.Error())
			}
			if !createAsstRsp.Result {
				return nil, params.Err.Error(createAsstRsp.Code)
			}

		}
	}

	if util.IsExistSupplierID(params.Header) {
		supplierID, err := util.GetSupplierID(params.Header)
		if err != nil {
			return nil, params.Err.Errorf(common.CCErrCommParamsNeedInt, common.BKSupplierIDField)
		}
		data[common.BKSupplierIDField] = supplierID
	}

	bizInst, err := b.inst.CreateInst(params, obj, data)
	if nil != err {
		blog.Errorf("[operation-biz] failed to create business, error info is %s", err.Error())
		return bizInst, err
	}

	bizID, err := bizInst.GetInstID()
	if nil != err {
		blog.Errorf("create business failed to create business, error info is %s", err.Error())
		return bizInst, params.Err.New(common.CCErrTopoAppCreateFailed, err.Error())
	}

	// register business to auth
	bizName, err := data.String(common.BKAppNameField)
	if err != nil {
		blog.Errorf("create business, but got invalid business name. err: %v", err)
		return bizInst, params.Err.New(common.CCErrTopoAppCreateFailed, err.Error())
	}

	if err := b.authManager.RegisterBusinessesByID(params.Context, params.Header, bizID); err != nil {
		blog.Errorf("create business: %s, but register business resource failed, err: %v", bizName, err)
		return bizInst, params.Err.New(common.CCErrCommRegistResourceToIAMFailed, err.Error())
	}

	// create set
	objSet, err := b.obj.FindSingleObject(params, common.BKInnerObjIDSet)
	if nil != err {
		blog.Errorf("failed to search the set, %s", err.Error())
		return nil, params.Err.New(common.CCErrTopoAppCreateFailed, err.Error())
	}

	setData := mapstr.New()
	setData.Set(common.BKAppIDField, bizID)
	setData.Set(common.BKInstParentStr, bizID)
	setData.Set(common.BKSetNameField, common.DefaultResSetName)
	setData.Set(common.BKDefaultField, common.DefaultResSetFlag)

	setInst, err := b.set.CreateSet(params, objSet, bizID, setData)
	if nil != err {
		blog.Errorf("create business failed to create business, error info is %s", err.Error())
		return bizInst, params.Err.New(common.CCErrTopoAppCreateFailed, err.Error())
	}

	setID, err := setInst.GetInstID()
	if nil != err {
		blog.Errorf("create business failed to create business, error info is %s", err.Error())
		return bizInst, params.Err.New(common.CCErrTopoAppCreateFailed, err.Error())
	}

	// create module
	objModule, err := b.obj.FindSingleObject(params, common.BKInnerObjIDModule)
	if nil != err {
		blog.Errorf("failed to search the set, %s", err.Error())
		return nil, params.Err.New(common.CCErrTopoAppCreateFailed, err.Error())
	}

	moduleData := mapstr.New()
	moduleData.Set(common.BKSetIDField, setID)
	moduleData.Set(common.BKInstParentStr, setID)
	moduleData.Set(common.BKAppIDField, bizID)
	moduleData.Set(common.BKModuleNameField, common.DefaultResModuleName)
	moduleData.Set(common.BKDefaultField, common.DefaultResModuleFlag)

	_, err = b.module.CreateModule(params, objModule, bizID, setID, moduleData)
	if nil != err {
		blog.Errorf("create business failed to create business, error info is %s", err.Error())
		return bizInst, params.Err.New(common.CCErrTopoAppCreateFailed, err.Error())
	}

	// create fault module
	faultModuleData := mapstr.New()
	faultModuleData.Set(common.BKSetIDField, setID)
	faultModuleData.Set(common.BKInstParentStr, setID)
	faultModuleData.Set(common.BKAppIDField, bizID)
	faultModuleData.Set(common.BKModuleNameField, common.DefaultFaultModuleName)
	faultModuleData.Set(common.BKDefaultField, common.DefaultFaultModuleFlag)

	_, err = b.module.CreateModule(params, objModule, bizID, setID, faultModuleData)
	if nil != err {
		blog.Errorf("create business failed to create business, error info is %s", err.Error())
		return bizInst, params.Err.New(common.CCErrTopoAppCreateFailed, err.Error())
	}

	return bizInst, nil
}

func (b *business) DeleteBusiness(params types.ContextParams, obj model.Object, bizID int64) error {
	if err := b.authManager.DeregisterBusinessByRawID(params.Context, params.Header, bizID); err != nil {
		blog.Errorf("delete business: %d, but deregister business from auth failed, err: %v", bizID, err)
		return params.Err.New(common.CCErrCommUnRegistResourceToIAMFailed, err.Error())
	}

	setObj, err := b.obj.FindSingleObject(params, common.BKInnerObjIDSet)
	if nil != err {
		blog.Errorf("failed to search the set, %s", err.Error())
		return err
	}

	bizObj, err := b.obj.FindSingleObject(params, common.BKInnerObjIDApp)
	if nil != err {
		blog.Errorf("failed to search the set, %s", err.Error())
		return err
	}

	if err = b.set.DeleteSet(params, setObj, bizID, nil); nil != err {
		blog.Errorf("[operation-biz] failed to delete the set, error info is %s", err.Error())
		return params.Err.New(common.CCErrTopoAppDeleteFailed, err.Error())
	}

	innerCond := condition.CreateCondition()
	innerCond.Field(common.BKAppIDField).Eq(bizID)

	return b.inst.DeleteInst(params, bizObj, innerCond, true)
}

var businessCache = sync.Map{}

func (b *business) FindBusiness(params types.ContextParams, obj model.Object, fields []string, cond condition.Condition) (count int, results []inst.Inst, err error) {
	var applist, cacheList []int64
	var autherr error
	var authC = make(chan struct{})

	if b.authManager.Enabled() {
		// it will take a while, so Let the Bullets Fly
		go func() {
			applist, autherr = b.authManager.Authorize.GetAuthorizedBusinessList(params.Context, authmeta.UserInfo{UserName: params.User, SupplierAccount: params.SupplierAccount})
			if autherr == nil {
				sort.Sort(util.Int64Slice(applist))
				businessCache.Store(params.SupplierAccount+":"+params.User, applist)
			}
			close(authC)
		}()
		if tmp, ok := businessCache.Load(params.SupplierAccount + ":" + params.User); ok {
			cacheList = tmp.([]int64)
		} else {
			<-authC
			cacheList = applist
		}
		cond.Field(common.BKAppIDField).In(cacheList)
	}

	query := &metadata.QueryInput{}
	cond.Field(common.BKDefaultField).Eq(0)
	query.Condition = cond.ToMapStr()
	query.Limit = int(cond.GetLimit())
	if query.Limit > 500 {
		query.Limit = 500
	}
	query.Fields = strings.Join(fields, ",")
	query.Sort = cond.GetSort()
	query.Start = int(cond.GetStart())
	count, results, err = b.inst.FindInst(params, obj, query, false)

	if !b.authManager.Enabled() {
		return count, results, err
	}
	<-authC
	if util.SliceInt64Equal(cacheList, applist) {
		return count, results, err
	}

	cond.Field(common.BKAppIDField).In(cacheList)
	query.Condition = cond.ToMapStr()
	return b.inst.FindInst(params, obj, query, false)
}

func (b *business) GetInternalModule(params types.ContextParams, obj model.Object, bizID int64) (count int, result *metadata.InnterAppTopo, err error) {

	// search the sets
	cond := condition.CreateCondition()
	cond.Field(common.BKAppIDField).Eq(bizID)
	cond.Field(common.BKDefaultField).Eq(common.DefaultResModuleFlag)
	setObj, err := b.obj.FindSingleObject(params, common.BKInnerObjIDSet)
	if nil != err {
		return 0, nil, params.Err.New(common.CCErrTopoAppSearchFailed, err.Error())
	}

	querySet := &metadata.QueryInput{}
	querySet.Condition = cond.ToMapStr()
	_, sets, err := b.set.FindSet(params, setObj, querySet)
	if nil != err {
		return 0, nil, params.Err.New(common.CCErrTopoAppSearchFailed, err.Error())
	}

	// search modules
	cond.Field(common.BKDefaultField).In([]int{
		common.DefaultResModuleFlag,
		common.DefaultFaultModuleFlag,
	})

	moduleObj, err := b.obj.FindSingleObject(params, common.BKInnerObjIDModule)
	if nil != err {
		return 0, nil, params.Err.New(common.CCErrTopoAppSearchFailed, err.Error())
	}

	queryModule := &metadata.QueryInput{}
	queryModule.Condition = cond.ToMapStr()
	_, modules, err := b.module.FindModule(params, moduleObj, queryModule)
	if nil != err {
		return 0, nil, params.Err.New(common.CCErrTopoAppSearchFailed, err.Error())
	}

	// construct result
	result = &metadata.InnterAppTopo{}
	for _, set := range sets {
		id, err := set.GetInstID()
		if nil != err {
			return 0, nil, params.Err.New(common.CCErrTopoAppSearchFailed, err.Error())
		}
		name, err := set.GetInstName()
		if nil != err {
			return 0, nil, params.Err.New(common.CCErrTopoAppSearchFailed, err.Error())
		}

		result.SetID = id
		result.SetName = name
		break // should be only one set
	}

	for _, module := range modules {
		id, err := module.GetInstID()
		if nil != err {
			return 0, nil, params.Err.New(common.CCErrTopoAppSearchFailed, err.Error())
		}
		name, err := module.GetInstName()
		if nil != err {
			return 0, nil, params.Err.New(common.CCErrTopoAppSearchFailed, err.Error())
		}

		result.Module = append(result.Module, metadata.InnerModule{
			ModuleID:   id,
			ModuleName: name,
		})
	}

	return 0, result, nil
}

func (b *business) UpdateBusiness(params types.ContextParams, data mapstr.MapStr, obj model.Object, bizID int64) error {
	if biz, exist := data.Get(common.BKAppNameField); exist {
		bizName, err := data.String(common.BKAppNameField)
		if err != nil {
			blog.Errorf("update business, but got invalid business name: %v, id: %d", biz, bizID)
			return params.Err.Error(common.CCErrCommParamsIsInvalid)
		}

		if err := b.authManager.UpdateRegisteredBusinessByID(params.Context, params.Header, bizID); err != nil {
			blog.Errorf("update business name: %s, but update resource to auth failed, err: %v", bizName, err)
			return params.Err.New(common.CCErrCommRegistResourceToIAMFailed, err.Error())
		}
	}

	innerCond := condition.CreateCondition()
	innerCond.Field(common.BKAppIDField).Eq(bizID)

	return b.inst.UpdateInst(params, data, obj, innerCond, bizID)
}
