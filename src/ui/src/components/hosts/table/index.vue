<template>
    <div class="hosts-table-layout">
        <div class="hosts-options">
            <slot name="options">
                <span class="inline-block-middle"
                    v-cursor="{
                        active: !$isAuthorized(editAuth),
                        auth: [editAuth]
                    }">
                    <bk-button class="options-button" type="primary"
                        :disabled="!table.checked.length || !$isAuthorized(editAuth)"
                        @click="handleMultipleEdit">
                        {{$t('Common["编辑"]')}}
                    </bk-button>
                </span>
                <span class="inline-block-middle"
                    v-cursor="{
                        active: !$isAuthorized(transferAuth),
                        auth: [transferAuth]
                    }">
                    <bk-button class="options-button" type="default"
                        :disabled="!table.checked.length || !$isAuthorized(transferAuth)"
                        @click="transfer.show = true">
                        {{$t('BusinessTopology["转移"]')}}
                    </bk-button>
                </span>
                <bk-button class="options-button" type="submit default"
                    form="exportForm"
                    :disabled="!table.checked.length">
                    {{$t('ModelManagement["导出"]')}}
                </bk-button>
                <form id="exportForm" :action="table.exportUrl" method="POST" hidden>
                    <input type="hidden" name="bk_host_id" :value="table.checked">
                    <input type="hidden" name="export_custom_fields"
                        v-if="usercustom[columnsConfigKey]"
                        :value="usercustom[columnsConfigKey]">
                    <input type="hidden" name="bk_biz_id" value="-1">
                    <input type="hidden" name="metadata"
                        v-if="$route.name !== 'resource'"
                        :value="JSON.stringify($injectMetadata().metadata)">
                </form>
                <cmdb-clipboard-selector class="options-button"
                    :list="clipboardList"
                    :disabled="!table.checked.length"
                    @on-copy="handleCopy">
                </cmdb-clipboard-selector>
                <bk-button class="options-button quick-search-button" type="default"
                    v-if="quickSearch"
                    ref="quickSearchButton"
                    @click="quickSearchStatus.active = true">
                    {{$t('HostResourcePool["筛选"]')}}
                    <i class="bk-icon icon-angle-down"></i>
                </bk-button>
                <div class="fr" v-tooltip="$t('BusinessTopology[\'列表显示属性配置\']')">
                    <bk-button class="options-button" type="default" style="margin-right: 0"
                        @click="columnsConfig.show = true">
                        <i class="icon-cc-setting"></i>
                    </bk-button>
                </div>
            </slot>
        </div>
        <cmdb-collapse-transition @after-enter="handleQuickSearchToggle" @after-leave="handleQuickSearchToggle">
            <cmdb-host-quick-search
                v-if="quickSearch && quickSearchStatus.active"
                :properties="properties.host"
                @on-toggle="quickSearchStatus.active = false"
                @on-search="handleQuickSearch">
            </cmdb-host-quick-search>
        </cmdb-collapse-transition>
        <cmdb-table class="hosts-table" ref="hostsTable"
            :loading="$loading()"
            :checked.sync="table.checked"
            :header="table.header"
            :list="table.list"
            :default-sort="table.defaultSort"
            :pagination.sync="table.pagination"
            :wrapper-minus-height="table.tableMinusHeight"
            @handleRowClick="handleRowClick"
            @handleSortChange="handleSortChange"
            @handlePageChange="handlePageChange"
            @handleSizeChange="handleSizeChange"
            @handleCheckAll="handleCheckAll">
            <template v-for="(header, index) in table.header" :slot="header.id" slot-scope="{ item }">
                <label class="table-checkbox bk-form-checkbox bk-checkbox-small"
                    :key="index"
                    v-if="header.id === 'bk_host_id'"
                    @click.stop>
                    <input type="checkbox"
                        :value="item['host']['bk_host_id']"
                        v-model="table.checked">
                </label>
                <span v-else :key="index">
                    {{getHostCellText(header, item)}}
                </span>
            </template>
        </cmdb-table>
        <cmdb-slider :is-show.sync="slider.show" :title="slider.title" :before-close="handleSliderBeforeClose">
            <bk-tab :active-name.sync="tab.active" slot="content">
                <bk-tabpanel name="attribute" :title="$t('Common[\'属性\']')" style="width: calc(100% + 40px);margin: 0 -20px;">
                    <cmdb-form-multiple v-if="tab.attribute.type === 'multiple'"
                        ref="multipleForm"
                        :properties="properties.host"
                        :property-groups="propertyGroups"
                        :object-unique="objectUnique"
                        :save-auth="saveAuth"
                        @on-submit="handleMultipleSave"
                        @on-cancel="handleMultipleCancel">
                    </cmdb-form-multiple>
                </bk-tabpanel>
            </bk-tab>
        </cmdb-slider>
        <cmdb-slider
            :is-show.sync="columnsConfig.show"
            :width="600"
            :title="$t('BusinessTopology[\'列表显示属性配置\']')">
            <cmdb-columns-config slot="content"
                :properties="columnsConfigProperties"
                :selected="columnsConfig.selected"
                :disabled-columns="columnsConfig.disabledColumns"
                @on-apply="handleApplyColumnsConfig"
                @on-cancel="columnsConfig.show = false"
                @on-reset="handleResetColumnsConfig">
            </cmdb-columns-config>
        </cmdb-slider>
        <bk-dialog
            :is-show.sync="transfer.show"
            :draggable="true"
            :close-icon="false"
            :has-footer="false"
            :has-header="false"
            :padding="0"
            :width="720">
            <div class="transfer-title" slot="tools">
                <i class="icon icon-cc-shift mr5"></i>
                <span>{{$t('Common[\'主机转移\']')}}</span>
                <span v-if="selectedHosts.length === 1">{{selectedHosts[0]['host']['bk_host_innerip']}}</span>
            </div>
            <div class="transfer-content" slot="content">
                <cmdb-transfer-host v-if="transfer.show"
                    :transfer-resource-auth="transferResourceAuth"
                    :selected-hosts="selectedHosts"
                    @on-success="handleTransferSuccess"
                    @on-cancel="transfer.show = false">
                </cmdb-transfer-host>
            </div>
        </bk-dialog>
    </div>
</template>

<script>
    import { mapGetters, mapActions } from 'vuex'
    // import cmdbHostsFilter from '@/components/hosts/filter'
    import cmdbColumnsConfig from '@/components/columns-config/columns-config'
    import cmdbTransferHost from '@/components/hosts/transfer'
    import cmdbHostQuickSearch from './_quick-search.vue'
    export default {
        components: {
            // cmdbHostsFilter,
            cmdbColumnsConfig,
            cmdbTransferHost,
            cmdbHostQuickSearch
        },
        props: {
            columnsConfigProperties: {
                type: Array,
                required: true
            },
            columnsConfigKey: {
                type: String,
                required: true
            },
            columnsConfigDisabledColumns: {
                type: Array,
                default () {
                    return ['bk_host_innerip', 'bk_cloud_id', 'bk_module_name']
                }
            },
            quickSearch: {
                type: Boolean,
                default: false
            },
            saveAuth: {
                type: [String, Array],
                default: ''
            },
            editAuth: {
                type: [String, Array],
                default: ''
            },
            deleteAuth: {
                type: [String, Array],
                default: ''
            },
            transferAuth: {
                type: [String, Array],
                default: ''
            },
            transferResourceAuth: {
                type: [String, Array],
                default: ''
            }
        },
        data () {
            return {
                objectUnique: [],
                properties: {
                    biz: [],
                    host: [],
                    set: [],
                    module: []
                },
                propertyGroups: [],
                quickSearchStatus: {
                    active: false
                },
                table: {
                    checked: [],
                    header: [],
                    list: [],
                    allList: [],
                    pagination: {
                        current: 1,
                        size: 10,
                        count: 0
                    },
                    defaultSort: 'bk_host_id',
                    sort: 'bk_host_id',
                    exportUrl: `${window.API_HOST}hosts/export`,
                    tableMinusHeight: 200
                },
                filter: {
                    business: '',
                    condition: {}
                },
                slider: {
                    show: false,
                    title: ''
                },
                tab: {
                    active: 'attribute',
                    attribute: {
                        type: 'details',
                        inst: {
                            details: {},
                            edit: {},
                            original: {}
                        }
                    }
                },
                columnsConfig: {
                    show: false,
                    selected: [],
                    disabledColumns: ['bk_host_innerip', 'bk_cloud_id', 'bk_module_name', 'bk_set_name']
                },
                transfer: {
                    show: false
                }
            }
        },
        computed: {
            ...mapGetters(['supplierAccount']),
            ...mapGetters('userCustom', ['usercustom']),
            customColumns () {
                return this.usercustom[this.columnsConfigKey] || []
            },
            clipboardList () {
                return this.table.header.filter(header => header.type !== 'checkbox')
            },
            selectedHosts () {
                return this.table.allList.filter(host => this.table.checked.includes(host['host']['bk_host_id']))
            }
        },
        watch: {
            'table.checked' (checked) {
                this.$emit('on-checked', checked)
            },
            'table.header' (header) {
                this.$emit('on-set-header', header)
            },
            'slider.show' (show) {
                if (!show) {
                    this.tab.active = 'attribute'
                }
            },
            customColumns () {
                this.setTableHeader()
            },
            columnsConfigProperties () {
                this.setTableHeader()
            }
        },
        async created () {
            try {
                await Promise.all([
                    this.getProperties(),
                    this.getHostPropertyGroups()
                ])
            } catch (e) {
                console.log(e)
            }
        },
        mounted () {
            this.calcTableMinusHeight()
        },
        methods: {
            ...mapActions('objectModelProperty', ['batchSearchObjectAttribute']),
            ...mapActions('objectModelFieldGroup', ['searchGroup']),
            ...mapActions('hostUpdate', ['updateHost']),
            ...mapActions('hostSearch', ['searchHost', 'searchHostByInnerip']),
            calcTableMinusHeight () {
                const $table = this.$refs.hostsTable.$el
                this.table.tableMinusHeight = $table.getBoundingClientRect().top + 20
            },
            getProperties () {
                return this.batchSearchObjectAttribute({
                    params: this.$injectMetadata({
                        bk_obj_id: { '$in': Object.keys(this.properties) },
                        bk_supplier_account: this.supplierAccount
                    }, { inject: this.$route.name !== 'resource' }),
                    config: {
                        requestId: `post_batchSearchObjectAttribute_${Object.keys(this.properties).join('_')}`,
                        requestGroup: Object.keys(this.properties).map(id => `post_searchObjectAttribute_${id}`)
                    }
                }).then(result => {
                    Object.keys(this.properties).forEach(objId => {
                        this.properties[objId] = result[objId]
                    })
                    return result
                })
            },
            getHostPropertyGroups () {
                return this.searchGroup({
                    objId: 'host',
                    params: this.$injectMetadata(),
                    config: {
                        fromCache: true,
                        requestId: 'post_searchGroup_host'
                    }
                }).then(groups => {
                    this.propertyGroups = groups
                    return groups
                })
            },
            setTableHeader () {
                const properties = this.$tools.getHeaderProperties(this.columnsConfigProperties, this.customColumns, this.columnsConfigDisabledColumns)
                this.table.header = [{
                    id: 'bk_host_id',
                    type: 'checkbox',
                    objId: 'host'
                }].concat(properties.map(property => {
                    return {
                        id: property['bk_property_id'],
                        name: property['bk_property_name'],
                        objId: property['bk_obj_id'],
                        sortable: property['bk_obj_id'] === 'host' && !['foreignkey'].includes(property['bk_property_type'])
                    }
                }))
                this.columnsConfig.selected = properties.map(property => property['bk_property_id'])
            },
            setAllHostList (list) {
                const newList = []
                list.forEach(item => {
                    const existItem = this.table.allList.find(existItem => existItem['host']['bk_host_id'] === item['host']['bk_host_id'])
                    if (existItem) {
                        Object.assign(existItem, item)
                    } else {
                        newList.push(item)
                    }
                })
                this.table.allList = [...this.table.allList, ...newList]
            },
            getHostCellText (header, item) {
                const objId = header.objId
                const propertyId = header.id
                const headerProperty = this.$tools.getProperty(this.properties[objId], propertyId)
                const originalValues = item[objId] instanceof Array ? item[objId] : [item[objId]]
                const text = []
                originalValues.forEach(value => {
                    const flatternedText = this.$tools.getPropertyText(headerProperty, value)
                    flatternedText ? text.push(flatternedText) : void (0)
                })
                return text.join(',') || '--'
            },
            getHostList () {
                this.searchHost({
                    params: {
                        ...this.filter.condition,
                        'bk_biz_id': this.filter.business,
                        page: {
                            start: (this.table.pagination.current - 1) * this.table.pagination.size,
                            limit: this.table.pagination.size,
                            sort: this.table.sort
                        }
                    },
                    config: {
                        requestId: 'searchHosts',
                        cancelPrevious: true
                    }
                }).then(data => {
                    this.table.pagination.count = data.count
                    this.table.list = data.info
                    this.setAllHostList(data.info)
                    return data
                }).catch(e => {
                    this.table.checked = []
                    this.table.list = []
                    this.table.pagination.count = 0
                })
            },
            getAllHostList () {
                return this.searchHost({
                    params: {
                        ...this.filter.condition,
                        'bk_biz_id': this.filter.business,
                        page: {}
                    },
                    config: {
                        requestId: 'searchAllHosts',
                        cancelPrevious: true
                    }
                }).then(data => {
                    this.table.allList = data.info
                    return data
                })
            },
            search (business, condition, resetPage = false) {
                this.filter.business = business
                this.filter.condition = condition
                if (resetPage) {
                    this.table.pagination.current = 1
                }
                this.getHostList()
            },
            handlePageChange (current) {
                this.table.pagination.current = current
                this.getHostList()
            },
            handleSizeChange (size) {
                this.table.pagination.size = size
                this.handlePageChange(1)
            },
            handleSortChange (sort) {
                this.table.sort = sort
                this.handlePageChange(1)
            },
            handleCopy (target) {
                const copyList = this.table.allList.filter(item => {
                    return this.table.checked.includes(item['host']['bk_host_id'])
                })
                const copyText = []
                this.$tools.clone(copyList).forEach(item => {
                    const cellText = this.getHostCellText(target, item)
                    if (cellText !== '--') {
                        copyText.push(cellText)
                    }
                })
                if (copyText.length) {
                    this.$copyText(copyText.join('\n')).then(() => {
                        this.$success(this.$t('Common["复制成功"]'))
                    }, () => {
                        this.$error(this.$t('Common["复制失败"]'))
                    })
                } else {
                    this.$info(this.$t('Common["该字段无可复制的值"]'))
                }
            },
            async handleCheckAll (type) {
                let list
                if (type === 'current') {
                    list = this.table.list
                } else {
                    const data = await this.getAllHostList()
                    list = data.info
                }
                this.table.checked = list.map(item => item['host']['bk_host_id'])
            },
            handleRowClick (item) {
                if (this.filter.business > -1) {
                    this.$router.push({
                        name: 'businessHostDetails',
                        params: {
                            business: this.filter.business,
                            id: item.host.bk_host_id
                        }
                    })
                } else {
                    this.$router.push({
                        name: 'resourceHostDetails',
                        params: {
                            id: item.host.bk_host_id
                        }
                    })
                }
            },
            batchUpdate (params) {
                return this.updateHost(params).then(data => {
                    this.$success(this.$t('Common[\'保存成功\']'))
                    this.getHostList()
                    return data
                })
            },
            async handleMultipleEdit () {
                this.objectUnique = await this.$store.dispatch('objectUnique/searchObjectUniqueConstraints', {
                    objId: 'host',
                    params: this.$injectMetadata({}, {
                        inject: this.$route.name !== 'resource'
                    })
                })
                this.tab.attribute.type = 'multiple'
                this.slider.title = this.$t('HostResourcePool[\'主机属性\']')
                this.slider.show = true
            },
            async handleMultipleSave (changedValues) {
                await this.batchUpdate(this.$injectMetadata({
                    ...changedValues,
                    'bk_host_id': this.table.checked.join(',')
                }, { inject: this.$route.name !== 'resource' }))
                this.slider.show = false
            },
            handleMultipleCancel () {
                this.slider.show = false
            },
            handleApplyColumnsConfig (properties) {
                this.$store.dispatch('userCustom/saveUsercustom', {
                    [this.columnsConfigKey]: properties.map(property => property['bk_property_id'])
                })
                this.columnsConfig.show = false
            },
            handleResetColumnsConfig () {
                this.$store.dispatch('userCustom/saveUsercustom', {
                    [this.columnsConfigKey]: []
                })
            },
            handleTransferSuccess () {
                this.table.checked = []
                this.transfer.show = false
                this.getHostList()
            },
            handleSliderBeforeClose () {
                if (this.tab.active === 'attribute' && this.tab.attribute.type !== 'details') {
                    const $form = this.tab.attribute.type === 'update' ? this.$refs.form : this.$refs.multipleForm
                    const changedValues = $form.changedValues
                    if (Object.keys(changedValues).length) {
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
                }
                return true
            },
            handleQuickSearchToggle () {
                this.calcTableMinusHeight()
            },
            handleQuickSearch (property, value, operator) {
                this.$emit('on-quick-search', property, value, operator)
            }
        }
    }
</script>

<style lang="scss" scoped>
    .hosts-options{
        font-size: 0;
        .options-button{
            position: relative;
            display: inline-block;
            vertical-align: middle;
            border-radius: 0;
            font-size: 14px;
            margin: 0 5px;
            &.quick-search-button {
                .icon-angle-down {
                    font-size: 12px;
                    top: 0;
                }
            }
            &:first-child {
                margin-left: 0;
            }
            &:hover{
                z-index: 1;
            }
        }
    }
    .hosts-table{
        margin-top: 20px;
    }
    .transfer-title{
        height: 50px;
        line-height: 50px;
        background-color: #f9f9f9;
        color: #333948;
        font-weight: bold;
        font-size: 14px;
        padding: 0 30px;
        border-bottom: 1px solid $cmdbBorderColor;
    }
    .transfer-content {
        height: 540px;
    }
</style>
