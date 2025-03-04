<template>
    <div class="api-wrapper" :style="{ 'padding-top': showFeatureTips ? '10px' : '' }">
        <feature-tips
            :feature-name="'customQuery'"
            :show-tips="showFeatureTips"
            :desc="$t('CustomQuery[\'动态分组提示\']')"
            :more-href="'https://docs.bk.tencent.com/cmdb/Introduction.html#%EF%BC%886%EF%BC%89%E5%8A%A8%E6%80%81%E5%88%86%E7%BB%84'"
            @close-tips="showFeatureTips = false">
        </feature-tips>
        <div class="filter-wrapper clearfix">
            <span class="inline-block-middle" v-cursor="{
                active: !$isAuthorized(OPERATION.C_CUSTOM_QUERY),
                auth: [OPERATION.C_CUSTOM_QUERY]
            }">
                <bk-button type="primary" class="api-btn"
                    :disabled="!$isAuthorized(OPERATION.C_CUSTOM_QUERY)"
                    @click="showUserAPISlider('create')">
                    {{$t("Common['新建']")}}
                </bk-button>
            </span>
            <div class="api-input fr">
                <input type="text" class="cmdb-form-input" :placeholder="$t('Inst[\'快速查询\']')" v-model="filter.name" @keyup.enter="getUserAPIList">
            </div>
        </div>
        <cmdb-table
            class="api-table"
            :loading="$loading('searchCustomQuery')"
            :header="table.header"
            :list="table.list"
            :pagination.sync="table.pagination"
            :wrapper-minus-height="220"
            @handlePageChange="handlePageChange"
            @handleSizeChange="handleSizeChange"
            @handleSortChange="handleSortChange"
            @handleRowClick="showUserAPIDetails">
            <template slot="create_time" slot-scope="{ item }">
                {{$tools.formatTime(item['create_time'])}}
            </template>
            <template slot="last_time" slot-scope="{ item }">
                {{$tools.formatTime(item['last_time'])}}
            </template>
            <div class="empty-info" slot="data-empty">
                <p>{{$t("Common['暂时没有数据']")}}</p>
            </div>
        </cmdb-table>
        <cmdb-slider
            :is-show.sync="slider.isShow"
            :has-quick-close="true"
            :width="430"
            :title="slider.title"
            :before-close="handleSliderBeforeClose">
            <v-define slot="content"
                ref="define"
                :id="slider.id"
                :biz-id="bizId"
                :type="slider.type"
                @delete="getUserAPIList"
                @create="handleCreate"
                @update="getUserAPIList"
                @cancel="hideUserAPISlider">
            </v-define>
        </cmdb-slider>
    </div>
</template>

<script>
    import { mapActions, mapGetters } from 'vuex'
    import featureTips from '@/components/feature-tips/index'
    import vDefine from './define'
    import { OPERATION } from './router.config.js'
    export default {
        components: {
            vDefine,
            featureTips
        },
        data () {
            return {
                showFeatureTips: false,
                OPERATION,
                filter: {
                    name: ''
                },
                table: {
                    header: [{
                        id: 'id',
                        name: 'ID'
                    }, {
                        id: 'name',
                        name: this.$t("CustomQuery['查询名称']")
                    }, {
                        id: 'create_user',
                        name: this.$t("CustomQuery['创建用户']")
                    }, {
                        id: 'create_time',
                        name: this.$t("CustomQuery['创建时间']")
                    }, {
                        id: 'modify_user',
                        name: this.$t("CustomQuery['修改人']")
                    }, {
                        id: 'last_time',
                        name: this.$t("CustomQuery['修改时间']")
                    }],
                    list: [],
                    sort: '-last_time',
                    defaultSort: '-last_time',
                    pagination: {
                        current: 1,
                        count: 0,
                        size: 10
                    }
                },
                slider: {
                    isShow: false,
                    isCloseConfirmShow: false,
                    type: 'create',
                    id: null,
                    title: this.$t("CustomQuery['新增查询']")
                }
            }
        },
        computed: {
            ...mapGetters(['featureTipsParams']),
            ...mapGetters('objectBiz', ['bizId']),
            searchParams () {
                const params = {
                    start: (this.table.pagination.current - 1) * this.table.pagination.size,
                    limit: this.table.pagination.size,
                    sort: this.table.sort
                }
                this.filter.name ? params['condition'] = { 'name': this.filter.name } : void (0)
                return params
            }
        },
        created () {
            this.$store.commit('setHeaderTitle', this.$t('Nav["动态分组"]'))
            this.showFeatureTips = this.featureTipsParams['customQuery']
            this.getUserAPIList()
        },
        methods: {
            ...mapActions('hostCustomApi', [
                'searchCustomQuery'
            ]),
            hideUserAPISlider () {
                this.slider.isShow = false
                this.slider.id = null
            },
            handleSliderBeforeClose () {
                if (this.$refs.define.isCloseConfirmShow()) {
                    return new Promise((resolve, reject) => {
                        this.$bkInfo({
                            title: this.$t('Common["退出会导致未保存信息丢失，是否确认？"]'),
                            confirmFn: () => {
                                resolve(true)
                            },
                            cancelFn: () => {
                                resolve(false)
                            }
                        })
                    })
                }
                return true
            },
            handleCreate (data) {
                this.slider.id = data['id']
                this.slider.type = 'update'
                this.slider.title = this.$t('CustomQuery["编辑查询"]')
                this.handlePageChange(1)
            },
            async getUserAPIList () {
                const res = await this.searchCustomQuery({
                    bizId: this.bizId,
                    params: this.searchParams,
                    config: {
                        requestId: 'searchCustomQuery'
                    }
                })
                if (res.count) {
                    this.table.list = res.info
                } else {
                    this.table.list = []
                }
                this.table.pagination.count = res.count
            },
            showUserAPISlider (type) {
                this.slider.isShow = true
                this.slider.type = type
                this.slider.title = this.$t('CustomQuery["新建查询"]')
            },
            /* 显示自定义API详情 */
            showUserAPIDetails (userAPI) {
                this.slider.isShow = true
                this.slider.type = 'update'
                this.slider.id = userAPI['id']
                this.slider.title = this.$t('CustomQuery["编辑查询"]')
            },
            handlePageChange (current) {
                this.table.pagination.current = current
                this.getUserAPIList()
            },
            handleSizeChange (size) {
                this.table.pagination.size = size
                this.handlePageChange(1)
            },
            handleSortChange (sort) {
                this.table.sort = sort
                this.getUserAPIList()
            }
        }
    }
</script>

<style lang="scss" scoped>
    .api-wrapper {
        .filter-wrapper {
            .business-selector {
                float: left;
                width: 170px;
                margin-right: 10px;
            }
            .api-btn {
                float: left;
            }
            .api-input {
                float: right;
                width: 320px;
            }
        }
        .api-table {
            margin-top: 20px;
        }
    }
</style>
