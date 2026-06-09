<template>
  <div class="p-6 space-y-6">
    <!-- Header -->
    <div class="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
      <div class="flex min-w-0 items-center gap-3">
        <CodexIcon :item="currentCodexRoute" size="lg" />
        <div class="min-w-0">
          <h1 class="text-2xl font-bold text-white">{{ pageTitle }}</h1>
          <p class="text-gray-400 text-sm mt-1">{{ pageSubtitle }}</p>
        </div>
      </div>
      <div class="stable-actions">
        <button @click="showImportDialog = true" class="btn btn-secondary header-action-btn" title="批量导入账号">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"/>
          </svg>
          导入
        </button>
        <button @click="openOAuthDialog" class="btn btn-secondary header-action-btn" title="OAuth 登录">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1"/>
          </svg>
          OAuth
        </button>
        <button @click="openAddAPIDialog" class="btn btn-primary header-action-btn" title="添加 API 账号">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
          </svg>
          API 账号
        </button>
        <button @click="openServiceConfig" class="btn btn-secondary header-action-btn" title="服务配置：代理池开关、对外 API Key、账号集合">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
          </svg>
          配置
        </button>
      </div>
    </div>

    <!-- Tab bar -->
    <div class="stable-tabs">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        @click="activeTab = tab.id"
        class="px-4 py-2 rounded-md text-sm font-medium transition-colors"
        :class="activeTab === tab.id ? 'bg-blue-600 text-white' : 'text-gray-400 hover:text-white'"
      >
        {{ tab.label }}
        <span v-if="tab.count > 0" class="ml-1.5 px-1.5 py-0.5 text-xs rounded-full" :class="activeTab === tab.id ? 'bg-blue-500' : 'bg-gray-700'">
          {{ tab.count }}
        </span>
      </button>
    </div>

    <!-- OAuth Accounts Tab -->
    <div v-if="activeTab === 'oauth'">
      <div v-if="loading" class="text-center py-12 text-gray-400">加载中...</div>
      <div v-else-if="oauthAccounts.length === 0" class="text-center py-12 text-gray-500">
        <p class="text-base mb-1">暂无 OAuth 账号</p>
        <p class="text-sm">点击"批量导入"或"OAuth 登录"添加账号</p>
      </div>
      <template v-else>
        <!-- Quota refresh bar -->
        <div class="mb-3 space-y-2">
          <div v-if="quotaLastFetched" class="text-xs text-gray-500">
            配额更新于 {{ quotaLastFetched }}
          </div>
          <div class="account-toolbar account-toolbar--oauth">
            <div class="toolbar-section toolbar-section--search">
              <input
                v-model="searchQuery"
                class="toolbar-input toolbar-search"
                placeholder="搜索账号"
                title="按账号标识或账号 ID 搜索"
              />
              <select
                v-model="planGroupFilter"
                @change="setPlanGroupFilter(planGroupFilter)"
                class="toolbar-select toolbar-select--plan"
                title="账号类型"
              >
                <option v-for="group in planGroups" :key="group.id" :value="group.id">
                  {{ group.id === 'all' ? group.label : `${group.label}（${group.count}）` }}
                </option>
              </select>
              <select
                v-model="accountSortMode"
                @change="setAccountSortMode(accountSortMode)"
                class="toolbar-select toolbar-select--sort"
                title="排序"
              >
                <option v-for="option in accountSortOptions" :key="option.id" :value="option.id">
                  {{ option.label }}
                </option>
              </select>
              <select
                v-model="quotaFilter"
                class="toolbar-select toolbar-select--quota"
                title="按上次配额查询结果过滤账号"
              >
                <option value="all">全部</option>
                <option value="200">200（成功）</option>
                <option value="401">401（失效/未授权）</option>
                <option value="403">403（地区受限/禁止）</option>
                <option value="429">429（限流）</option>
                <option value="503">503（服务不可用）</option>
              </select>
            </div>

            <div class="toolbar-section toolbar-section--view">
              <button
                @click="toggleAccountLayout"
                class="toolbar-btn toolbar-btn--layout"
                :class="accountLayout === 'dense' ? 'bg-blue-600/20 hover:bg-blue-600/30 border-blue-500/40 text-blue-200' : 'bg-gray-800 hover:bg-gray-700 border-gray-700 text-gray-300'"
                :title="accountLayout === 'dense' ? '当前为紧凑布局：点击切换标准布局' : '当前为标准布局：点击切换紧凑布局'"
                aria-label="切换账号列表布局"
              >
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4h6v6H4V4zm10 0h6v6h-6V4zM4 14h6v6H4v-6zm10 0h6v6h-6v-6z"/>
                </svg>
                {{ accountLayout === 'dense' ? '紧凑' : '标准' }}
              </button>
              <button
                @click="toggleAccountPrivacy"
                class="toolbar-btn toolbar-btn--layout"
                :class="hideAccountEmails ? 'bg-sky-600/20 hover:bg-sky-600/30 border-sky-500/40 text-sky-200' : 'bg-gray-800 hover:bg-gray-700 border-gray-700 text-gray-300'"
                :title="hideAccountEmails ? '隐私模式已开启：点击显示邮箱' : '隐私模式：点击隐藏账号邮箱'"
                aria-label="切换账号邮箱隐私显示"
              >
                <svg v-if="hideAccountEmails" class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21"/>
                </svg>
                <svg v-else class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/>
                </svg>
                {{ hideAccountEmails ? '隐私开' : '隐私' }}
              </button>
            </div>

            <div class="toolbar-section toolbar-section--actions">
              <button
                @click="refreshAllTokens"
                :disabled="refreshingAllTokens || oauthAccounts.length === 0"
                class="toolbar-btn toolbar-btn-neutral"
                title="刷新全部 OAuth 账号的 Token，全部完成并落库后自动导出最新 JSON"
              >
                <svg class="w-3.5 h-3.5" :class="refreshingAllTokens ? 'animate-spin' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
                </svg>
                {{ refreshingAllTokens ? '刷新中' : '刷新Token' }}
              </button>
              <button
                @click="fetchAllQuotas"
                :disabled="fetchingQuotas || oauthAccounts.length === 0"
                class="toolbar-btn toolbar-btn-neutral"
                title="查询全部 OAuth 账号配额"
              >
                <svg class="w-3.5 h-3.5" :class="fetchingQuotas ? 'animate-spin' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
                </svg>
                {{ fetchingQuotas ? '查询中' : '配额' }}
              </button>
              <button
                @click="exportAccounts"
                :disabled="exportingAccounts"
                class="toolbar-btn toolbar-btn-success"
                title="导出全部账号的最新落库数据"
              >
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/>
                </svg>
                {{ exportingAccounts ? '导出中' : '导出' }}
              </button>
            </div>

            <div class="toolbar-section toolbar-section--selection">
              <button
                v-if="filteredOAuthAccounts.length > 0"
                @click="toggleSelectAllOAuth"
                :disabled="bulkDeleting"
                class="toolbar-btn toolbar-btn-neutral toolbar-btn--select"
                :title="allFilteredOAuthSelected ? `取消全选当前筛选结果（${filteredOAuthAccounts.length}）` : `全选当前筛选结果（${filteredOAuthAccounts.length}）`"
              >
                {{ allFilteredOAuthSelected ? '取消' : `全选 ${filteredOAuthAccounts.length}` }}
              </button>
              <button
                v-if="selectedOAuthIds.length > 0"
                @click="clearOAuthSelection"
                :disabled="bulkDeleting"
                class="toolbar-btn toolbar-btn-neutral"
                title="清空当前已选 OAuth 账号"
              >
                清空
              </button>
              <div v-if="filteredOAuthAccounts.length > 0" class="toolbar-status">
                已选 {{ selectedOAuthIds.length }}
              </div>
              <button
                v-if="selectedOAuthIds.length > 0"
                @click="openBulkDeleteConfirm"
                :disabled="bulkDeleting"
                class="toolbar-btn toolbar-btn-danger"
                :title="`批量删除已选 OAuth 账号（${selectedOAuthIds.length}）`"
              >
                {{ bulkDeleting ? '删除中' : `删除 ${selectedOAuthIds.length}` }}
              </button>
            </div>
          </div>
        </div>
        <div class="grid" :class="accountGridClass">
          <div
            v-for="account in paginatedOAuth"
            :key="account.id"
            class="account-card-compact"
            :class="[
              account.is_codex_active ? 'ring-1 ring-blue-500/60' : '',
              isOAuthSelected(account.id) ? 'account-card-compact--selected' : '',
              accountLayout === 'dense' ? 'account-card-compact--dense' : '',
            ]"
          >
            <!-- Row 1: account identity + Codex badge -->
            <div class="flex items-center gap-2 min-w-0 mb-2">
              <label class="selection-checkbox shrink-0" :title="isOAuthSelected(account.id) ? '取消选择' : '选择该账号'">
                <input
                  type="checkbox"
                  :checked="isOAuthSelected(account.id)"
                  @change="toggleOAuthSelection(account.id)"
                />
                <span></span>
              </label>
              <span class="inline-block w-2 h-2 rounded-full shrink-0" :class="account.proxy_enabled ? 'bg-green-400' : 'bg-gray-500'"></span>
              <span class="text-sm font-medium text-white truncate flex-1" :title="accountDisplayTitle(account)">
                {{ accountDisplayLabel(account) }}
              </span>
              <span v-if="account.is_codex_active" class="shrink-0 text-[10px] font-bold text-blue-300 bg-blue-600/30 px-1.5 py-0.5 rounded">Codex</span>
              <span v-if="account.status === 'reauth_required'" class="shrink-0 text-[10px] font-bold text-red-300 bg-red-600/20 px-1.5 py-0.5 rounded">重登</span>
              <span v-if="account._quota_http_status && account._quota_http_status !== 200" class="quota-status-badge shrink-0 text-[10px] font-bold px-1.5 py-0.5 rounded" :class="quotaStatusBadgeClass(account._quota_http_status)">{{ account._quota_http_status }}</span>
            </div>
            <div v-if="accountLayout !== 'dense' && (accountGroupNames(account).length || account.tag_name)" class="flex flex-wrap gap-1 mb-2">
              <span v-for="name in accountGroupNames(account)" :key="name" class="text-[10px] px-1.5 py-0.5 rounded bg-indigo-500/15 text-indigo-200">
                {{ name }}
              </span>
              <span
                v-if="account.tag_name"
                class="text-[10px] px-1.5 py-0.5 rounded text-white"
                :style="{ backgroundColor: account.tag_color || '#4B5563' }"
              >
                {{ account.tag_name }}
              </span>
            </div>
            <!-- Row 2: info + quota -->
            <div class="flex items-center gap-3 text-[11px] text-gray-500 min-w-0" :class="accountLayout === 'dense' ? 'mb-1' : 'mb-1.5'">
              <span v-if="account.expires_at" class="truncate"
                :class="isExpired(account.expires_at) ? 'text-red-400' : isExpiringSoon(account.expires_at) ? 'text-yellow-400' : ''">
                {{ formatDate(account.expires_at) }}
                <span v-if="isExpired(account.expires_at)" class="text-red-400 ml-0.5">过期</span>
              </span>
              <span v-if="account.chatgpt_account_id" class="truncate font-mono">{{ account.chatgpt_account_id.slice(0, 12) }}</span>
            </div>
            <!-- Row 2.5: plan badge + quota bars -->
            <div v-if="accountLayout !== 'dense'" class="mb-2 space-y-1">
              <!-- Plan type from JWT -->
              <div class="flex items-center gap-2">
                <span
                  v-if="planBadge(account)"
                  class="text-[9px] font-bold px-1.5 py-0.5 rounded uppercase tracking-wide"
                  :class="planBadge(account).cls"
                >{{ planBadge(account).text }}</span>
                <span v-if="isRegionRestricted(account)" class="text-[10px] text-amber-300">地区受限</span>
                <span v-if="account._verified && !hasDisplayQuotaData(account)" class="text-[10px] text-green-400">✓ 有效</span>
              </div>

              <!-- Forbidden badge -->
              <div v-if="account.quota_is_forbidden" class="flex items-center gap-1 rounded bg-red-500/10 px-2 py-1 text-[10px] text-red-400">
                <svg class="w-3 h-3" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.42 0-8-3.58-8-8 0-1.85.63-3.55 1.69-4.9L16.9 18.31C15.55 19.37 13.85 20 12 20zm6.31-3.1L7.1 5.69C8.45 4.63 10.15 4 12 4c4.42 0 8 3.58 8 8 0 1.85-.63 3.55-1.69 4.9z"/>
                </svg>
                账号被禁用
              </div>

              <div v-if="isRegionRestricted(account)" class="flex items-center gap-1 rounded bg-amber-500/10 px-2 py-1 text-[10px] text-amber-300">
                <svg class="w-3 h-3" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12 2 1 21h22L12 2zm0 6 1 7h-2l1-7zm0 10.5a1.25 1.25 0 1 1 0-2.5 1.25 1.25 0 0 1 0 2.5z"/>
                </svg>
                当前出口地区受限，无法刷新该账号
              </div>

              <!-- 5h quota bar -->
              <template v-if="!account.quota_is_forbidden && shouldShow5hQuota(account)">
                <div class="flex items-center gap-2">
                  <span class="shrink-0 text-[9px] text-gray-500 w-6">5h</span>
                  <div class="flex-1 bg-gray-700 rounded-full h-1.5 overflow-hidden">
                    <div
                      class="h-1.5 rounded-full transition-all duration-500"
                      :class="account.quota_5h_used_percent != null ? pctBarClass(100 - account.quota_5h_used_percent) : 'bg-gray-600/60'"
                      :style="{ width: account.quota_5h_used_percent != null ? (100 - account.quota_5h_used_percent) + '%' : '100%' }"
                    ></div>
                  </div>
                  <span class="shrink-0 text-[10px] font-semibold tabular-nums w-12 text-right" :class="account.quota_5h_used_percent != null ? pctColor(100 - account.quota_5h_used_percent) : 'text-gray-500'">
                    {{ account.quota_5h_used_percent != null ? `${Math.round(100 - account.quota_5h_used_percent)}%` : '未返回' }}
                  </span>
                </div>
                <div class="flex items-center justify-between text-[9px] pl-8">
                  <span v-if="account.quota_5h_reset_seconds" class="text-gray-600">重置: {{ formatResetTime(account.quota_5h_reset_seconds) }}</span>
                  <span v-else class="text-gray-600">窗口未返回</span>
                </div>
              </template>

              <!-- 7d quota bar -->
              <template v-if="!account.quota_is_forbidden && shouldShow7dQuota(account)">
                <div class="flex items-center gap-2">
                  <span class="shrink-0 text-[9px] text-gray-500 w-6">7d</span>
                  <div class="flex-1 bg-gray-700 rounded-full h-1.5 overflow-hidden">
                    <div
                      class="h-1.5 rounded-full transition-all duration-500"
                      :class="account.quota_7d_used_percent != null ? pctBarClass(100 - account.quota_7d_used_percent) : 'bg-gray-600/60'"
                      :style="{ width: account.quota_7d_used_percent != null ? (100 - account.quota_7d_used_percent) + '%' : '100%' }"
                    ></div>
                  </div>
                  <span class="shrink-0 text-[10px] font-semibold tabular-nums w-12 text-right" :class="account.quota_7d_used_percent != null ? pctColor(100 - account.quota_7d_used_percent) : 'text-gray-500'">
                    {{ account.quota_7d_used_percent != null ? `${Math.round(100 - account.quota_7d_used_percent)}%` : '未返回' }}
                  </span>
                </div>
                <div class="flex items-center justify-between text-[9px] pl-8">
                  <span v-if="account.quota_7d_reset_seconds" class="text-gray-600">重置: {{ formatResetTime(account.quota_7d_reset_seconds) }}</span>
                  <span v-else class="text-gray-600">窗口未返回</span>
                </div>
              </template>

              <!-- Legacy: old total/used format (backward compat) -->
              <template v-if="!account.quota_is_forbidden && account.quota_7d_used_percent == null && account.quota_5h_used_percent == null && account.quota_total">
                <div class="flex items-center gap-2">
                  <span class="shrink-0 text-[9px] text-gray-500 w-6">7d</span>
                  <div class="flex-1 bg-gray-700 rounded-full h-1.5 overflow-hidden">
                    <div
                      class="h-1.5 rounded-full transition-all duration-500"
                      :class="quotaBarClass(account)"
                      :style="{ width: quotaPct(account) + '%' }"
                    ></div>
                  </div>
                  <span class="shrink-0 text-[10px] font-semibold tabular-nums w-8 text-right" :class="quotaColor(account)">
                    {{ quotaPct(account) }}%
                  </span>
                </div>
                <div class="flex items-center justify-between text-[9px] pl-8">
                  <span class="text-gray-500">
                    已用 {{ account.quota_used ?? 0 }} / {{ account.quota_total }}
                  </span>
                </div>
              </template>

              <!-- Updated time -->
              <div v-if="account.quota_updated_at && hasDisplayQuotaData(account)" class="text-[9px] text-gray-600 text-right">
                {{ formatQuotaTime(account.quota_updated_at) }}
              </div>

              <!-- No quota data yet -->
              <div v-if="!hasDisplayQuotaData(account) && !account.quota_is_forbidden" class="text-[9px] text-gray-600">
                <span v-if="accountPlanType(account) === 'free'">免费账号·配额头部不开放</span>
                <span v-else>点击卡片「配额」或上方「查询配额」获取</span>
              </div>
            </div>
            <div v-else class="dense-account-meta">
              <span v-if="planBadge(account)" class="dense-pill" :class="planBadge(account).cls">{{ planBadge(account).text }}</span>
              <span v-if="account.quota_is_forbidden" class="dense-pill bg-red-500/15 text-red-300">禁用</span>
              <span v-else-if="shouldShow5hQuota(account)" class="dense-pill" :class="pctColor(100 - account.quota_5h_used_percent)">5h {{ Math.round(100 - account.quota_5h_used_percent) }}%</span>
              <span v-if="shouldShow7dQuota(account)" class="dense-pill" :class="pctColor(100 - account.quota_7d_used_percent)">7d {{ Math.round(100 - account.quota_7d_used_percent) }}%</span>
              <span v-if="isRegionRestricted(account)" class="dense-pill bg-amber-500/15 text-amber-300">地区</span>
              <span v-if="!planBadge(account) && !hasDisplayQuotaData(account) && !account.quota_is_forbidden" class="text-gray-600 truncate">未查配额</span>
            </div>
            <!-- Row 3: all action buttons in one row -->
            <div v-if="accountLayout !== 'dense'" class="card-actions">
              <button
                @click="toggleProxy(account)" :disabled="togglingProxyId === account.id"
                class="card-btn card-btn--text" :class="account.proxy_enabled ? 'card-btn--on' : 'card-btn--off'"
                :title="account.proxy_enabled ? '移出代理池' : '加入代理池'"
              >{{ togglingProxyId === account.id ? '...' : account.proxy_enabled ? '代理' : '代理' }}</button>
              <button
                @click="switchAccount(account)" :disabled="switchingId === account.id"
                class="card-btn card-btn--primary card-btn--text"
                title="切换到该账号"
              >{{ switchingId === account.id ? '...' : '切换' }}</button>
              <button
                @click="refreshToken(account)" :disabled="refreshingId === account.id"
                class="card-btn card-btn--secondary card-btn--icon" title="刷新 Token"
              >
                <svg class="w-3 h-3" :class="refreshingId === account.id ? 'animate-spin' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
                </svg>
              </button>
              <button
                @click="fetchQuotaForAccount(account)"
                :disabled="fetchingQuotas || isFetchingQuota(account.id)"
                class="card-btn card-btn--secondary card-btn--text"
                title="查询配额"
              >{{ isFetchingQuota(account.id) ? '...' : '配额' }}</button>
              <button @click="deleteAccount(account.id)" class="card-btn card-btn--danger card-btn--icon" title="删除">
                <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                </svg>
              </button>
            </div>
          </div>
        </div>
        <!-- Pagination -->
        <div v-if="filteredOAuthAccounts.length > 0" class="pagination-bar">
          <label class="pagination-page-size">
            <span>每页</span>
            <select :value="currentPageSize" class="pagination-page-size-select" @change="setAccountPageSize($event.target.value)">
              <option v-for="size in pageSizeOptions" :key="size" :value="size">{{ size }}</option>
            </select>
            <span>个</span>
          </label>
          <div class="pagination-nav">
            <button @click="oauthPage = Math.max(1, oauthPage - 1)" :disabled="oauthPage <= 1" class="btn btn-sm btn-secondary">上一页</button>
            <span class="text-gray-400">{{ oauthPage }} / {{ oauthTotalPages }}<span class="text-gray-600 ml-2">({{ oauthPaginationRangeText }})</span></span>
            <button @click="oauthPage = Math.min(oauthTotalPages, oauthPage + 1)" :disabled="oauthPage >= oauthTotalPages" class="btn btn-sm btn-secondary">下一页</button>
          </div>
        </div>
      </template>
    </div>

    <!-- API Accounts Tab -->
    <div v-if="activeTab === 'api'">
      <div v-if="apiAccounts.length === 0" class="text-center py-12 text-gray-500">
        <p class="text-base mb-1">暂无 API 账号</p>
        <p class="text-sm">点击「添加 API 账号」配置自定义 API 端点</p>
      </div>
      <template v-else>
        <div class="account-toolbar account-toolbar--api mb-3">
          <div class="toolbar-section toolbar-section--search">
            <input
              v-model="apiSearchQuery"
              class="toolbar-input toolbar-search"
              placeholder="搜索 API"
              title="按 provider、model、base URL 搜索 API 账号"
            />
          </div>
          <div class="toolbar-section toolbar-section--view">
            <button
              @click="toggleAccountLayout"
              class="toolbar-btn toolbar-btn--layout"
              :class="accountLayout === 'dense' ? 'bg-blue-600/20 hover:bg-blue-600/30 border-blue-500/40 text-blue-200' : 'bg-gray-800 hover:bg-gray-700 border-gray-700 text-gray-300'"
              :title="accountLayout === 'dense' ? '当前为紧凑布局：点击切换标准布局' : '当前为标准布局：点击切换紧凑布局'"
              aria-label="切换账号列表布局"
            >
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4h6v6H4V4zm10 0h6v6h-6V4zM4 14h6v6H4v-6zm10 0h6v6h-6v-6z"/>
              </svg>
              {{ accountLayout === 'dense' ? '紧凑' : '标准' }}
            </button>
          </div>
          <div class="toolbar-section toolbar-section--selection">
            <button
              v-if="filteredAPIAccounts.length > 0"
              @click="toggleSelectAllAPI"
              :disabled="bulkDeleting"
              class="toolbar-btn toolbar-btn-neutral toolbar-btn--select"
              :title="allAPISelected ? `取消全选当前 API 筛选结果（${filteredAPIAccounts.length}）` : `全选当前 API 筛选结果（${filteredAPIAccounts.length}）`"
            >
              {{ allAPISelected ? '取消' : `全选 ${filteredAPIAccounts.length}` }}
            </button>
            <button
              v-if="selectedAPIIds.length > 0"
              @click="clearAPISelection"
              :disabled="bulkDeleting"
              class="toolbar-btn toolbar-btn-neutral"
              title="清空当前已选 API 账号"
            >
              清空
            </button>
            <div class="toolbar-status">
              已选 {{ selectedAPIIds.length }}
            </div>
            <button
              v-if="selectedAPIIds.length > 0"
              @click="openBulkDeleteConfirm"
              :disabled="bulkDeleting"
              class="toolbar-btn toolbar-btn-danger"
              :title="`批量删除已选 API 账号（${selectedAPIIds.length}）`"
            >
              {{ bulkDeleting ? '删除中' : `删除 ${selectedAPIIds.length}` }}
            </button>
          </div>
        </div>
        <div class="grid" :class="accountGridClass">
          <div
            v-for="account in paginatedAPI"
            :key="account.id"
            class="account-card-compact account-card-compact--api"
            :class="[isAPISelected(account.id) ? 'account-card-compact--selected' : '', accountLayout === 'dense' ? 'account-card-compact--dense' : '']"
          >
            <!-- Row 1: provider -->
            <div class="flex items-center gap-2 min-w-0 mb-2">
              <label class="selection-checkbox shrink-0" :title="isAPISelected(account.id) ? '取消选择' : '选择该账号'">
                <input
                  type="checkbox"
                  :checked="isAPISelected(account.id)"
                  @change="toggleAPISelection(account.id)"
                />
                <span></span>
              </label>
              <span class="text-sm font-medium text-white truncate flex-1" :title="account.model_provider">{{ providerDisplayName(account.model_provider) }}</span>
              <span v-if="account.is_codex_active" class="shrink-0 text-[10px] font-bold text-blue-300 bg-blue-600/30 px-1.5 py-0.5 rounded">Codex</span>
              <span v-if="account.model" class="shrink-0 text-[10px] font-mono text-emerald-300 bg-emerald-600/20 px-1.5 py-0.5 rounded truncate max-w-[100px]">{{ account.model }}</span>
            </div>
            <!-- Row 2: info -->
            <div class="flex items-center gap-3 text-[11px] text-gray-500 mb-2.5 truncate">
              <span v-if="account.base_url" class="truncate font-mono">{{ account.base_url }}</span>
              <span v-if="account.wire_api" class="shrink-0">{{ account.wire_api }}</span>
            </div>
            <!-- Row 3: action buttons -->
            <div v-if="accountLayout !== 'dense'" class="card-actions">
              <button @click="switchAPIAccount(account)" :disabled="switchingId === account.id" class="card-btn card-btn--primary card-btn--text flex-1" title="切换配置">
                {{ switchingId === account.id ? '...' : '切换' }}
              </button>
              <button @click="testAPIAccount(account)" :disabled="testingAPIId === account.id" class="card-btn card-btn--secondary card-btn--text" title="测活：发送最小请求验证 Key 是否可用">
                {{ testingAPIId === account.id ? '...' : '测活' }}
              </button>
              <button @click="editAPIAccount(account)" class="card-btn card-btn--secondary card-btn--icon" title="编辑">
                <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
                </svg>
              </button>
              <button @click="deleteAccount(account.id)" class="card-btn card-btn--danger card-btn--icon" title="删除">
                <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                </svg>
              </button>
            </div>
          </div>
        </div>
        <!-- Pagination -->
        <div v-if="filteredAPIAccounts.length > 0" class="pagination-bar">
          <label class="pagination-page-size">
            <span>每页</span>
            <select :value="currentPageSize" class="pagination-page-size-select" @change="setAccountPageSize($event.target.value)">
              <option v-for="size in pageSizeOptions" :key="size" :value="size">{{ size }}</option>
            </select>
            <span>个</span>
          </label>
          <div class="pagination-nav">
            <button @click="apiPage = Math.max(1, apiPage - 1)" :disabled="apiPage <= 1" class="btn btn-sm btn-secondary">上一页</button>
            <span class="text-gray-400">{{ apiPage }} / {{ apiTotalPages }}<span class="text-gray-600 ml-2">({{ apiPaginationRangeText }})</span></span>
            <button @click="apiPage = Math.min(apiTotalPages, apiPage + 1)" :disabled="apiPage >= apiTotalPages" class="btn btn-sm btn-secondary">下一页</button>
          </div>
        </div>
      </template>
    </div>

    <!-- ==================== Modals ==================== -->

    <!-- Delete Confirm Dialog -->
    <div v-if="showDeleteConfirm" class="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4" @click.self="closeDeleteConfirm">
      <div class="bg-gray-900 border border-gray-700 rounded-2xl w-full max-w-md shadow-2xl">
        <div class="flex items-center justify-between p-6 border-b border-gray-700">
          <h2 class="text-lg font-semibold text-white">确认删除</h2>
          <button @click="closeDeleteConfirm" class="text-gray-400 hover:text-white" :disabled="deletingAccount">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>
        <div class="p-6 space-y-3">
          <div class="text-sm text-gray-200">
            将永久删除该账号<span v-if="deleteTargetLabel" class="text-white font-medium">（{{ deleteTargetLabel }}）</span>，此操作不可恢复。
          </div>
          <div class="text-xs text-gray-500">
            提示：删除后不会影响你本地浏览器/客户端已存在的 token 文件，只会从 EasyLLM 中移除该账号记录。
          </div>
        </div>
        <div class="p-6 pt-0 flex items-center justify-end gap-2">
          <button @click="closeDeleteConfirm" class="btn btn-secondary" :disabled="deletingAccount">取消</button>
          <button @click="confirmDeleteAccount" class="btn btn-danger" :disabled="deletingAccount">
            {{ deletingAccount ? '删除中...' : '删除' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Bulk Delete Confirm Dialog -->
    <div v-if="showBulkDeleteConfirm" class="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4" @click.self="closeBulkDeleteConfirm">
      <div class="bg-gray-900 border border-gray-700 rounded-2xl w-full max-w-lg shadow-2xl">
        <div class="flex items-center justify-between p-6 border-b border-gray-700">
          <h2 class="text-lg font-semibold text-white">确认批量删除</h2>
          <button @click="closeBulkDeleteConfirm" class="text-gray-400 hover:text-white" :disabled="bulkDeleting">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>
        <div class="p-6 space-y-3">
          <div class="text-sm text-gray-200">
            将永久删除已选中的 <span class="text-white font-semibold">{{ bulkDeleteIds.length }}</span> 个 {{ bulkDeleteAccountLabel }}，此操作不可恢复。
          </div>
          <div v-if="bulkDeleteScopeLabel" class="text-xs text-gray-400">
            选择范围：{{ bulkDeleteScopeLabel }}
          </div>
          <div v-if="bulkDeleteAllSelected" class="text-xs text-red-300 bg-red-600/10 border border-red-600/30 rounded-lg px-3 py-2">
            警告：你当前已全选 {{ bulkDeleteAccountLabel }}。
          </div>
          <div class="text-xs text-gray-500">
            提示：仅删除 EasyLLM 内的账号记录，不会删除你本地浏览器/客户端已存在的 token 文件。
          </div>
          <div v-if="bulkDeletePreview.length" class="bg-gray-800/60 border border-gray-700 rounded-lg p-3">
            <div class="text-[11px] text-gray-400 mb-1">将删除（预览前 {{ bulkDeletePreview.length }} 个）：</div>
            <div class="space-y-1 max-h-24 overflow-y-auto">
              <div v-for="(e, i) in bulkDeletePreview" :key="i" class="text-xs text-gray-300 truncate">
                {{ i + 1 }}. {{ e }}
              </div>
            </div>
          </div>
        </div>
        <div class="p-6 pt-0 flex items-center justify-end gap-2">
          <button @click="closeBulkDeleteConfirm" class="btn btn-secondary" :disabled="bulkDeleting">取消</button>
          <button @click="confirmBulkDelete" class="btn btn-danger" :disabled="bulkDeleting">
            {{ bulkDeleting ? '删除中...' : `删除 ${bulkDeleteIds.length} 个` }}
          </button>
        </div>
      </div>
    </div>

    <!-- Group Manager Dialog -->
    <div v-if="showGroupManager" class="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4" @click.self="showGroupManager = false">
      <div class="bg-gray-900 border border-gray-700 rounded-2xl w-full max-w-2xl shadow-2xl">
        <div class="flex items-center justify-between p-6 border-b border-gray-700">
          <h2 class="text-lg font-semibold text-white">账号分组</h2>
          <button @click="showGroupManager = false" class="text-gray-400 hover:text-white">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>
        <div class="p-6 space-y-4">
          <div class="grid sm:grid-cols-[1fr_auto] gap-2">
            <input v-model="newGroupName" class="input" placeholder="新分组名称" />
            <button @click="createAccountGroup" class="btn btn-primary">新建分组</button>
          </div>
          <div class="space-y-2 max-h-80 overflow-y-auto">
            <div v-for="group in accountGroups" :key="group.id" class="rounded-xl border border-gray-700 bg-gray-800/60 p-3">
              <div class="flex items-center justify-between gap-3">
                <div class="min-w-0">
                  <div class="font-medium text-white truncate">{{ group.name }}</div>
                  <div class="text-xs text-gray-500 mt-0.5">{{ group.account_ids.length }} 个账号</div>
                </div>
                <div class="flex gap-2 shrink-0">
                  <button
                    @click="addSelectedToGroup(group.id)"
                    :disabled="selectedOAuthIds.length === 0"
                    class="btn btn-xs btn-secondary"
                    title="把当前勾选账号加入这个分组"
                  >加入已选</button>
                  <button
                    @click="removeSelectedFromGroup(group.id)"
                    :disabled="selectedOAuthIds.length === 0"
                    class="btn btn-xs btn-secondary"
                    title="把当前勾选账号从这个分组移除"
                  >移出已选</button>
                  <button @click="deleteAccountGroup(group.id)" class="btn btn-xs btn-danger">删除</button>
                </div>
              </div>
            </div>
            <div v-if="accountGroups.length === 0" class="text-sm text-gray-500 text-center py-8">还没有分组</div>
          </div>
        </div>
      </div>
    </div>

    <!-- Batch Import Dialog -->
    <div v-if="showImportDialog" class="import-dialog-overlay fixed inset-0 flex items-center justify-center z-50 p-4">
      <div class="import-dialog-panel bg-gray-900 border border-gray-700 rounded-2xl w-full max-w-3xl shadow-2xl">
        <div class="flex items-center justify-between p-6 border-b border-gray-700">
          <h2 class="text-lg font-semibold text-white">批量导入账号</h2>
          <button @click="closeImportDialog" class="text-gray-400 hover:text-white">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>
        <div class="p-6 space-y-4">

          <!-- Import mode tabs -->
          <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-7 gap-1 bg-gray-800 rounded-lg p-1">
            <button
              v-for="m in importModes"
              :key="m.id"
              @click="selectImportMode(m.id)"
              class="min-w-0 px-2 py-1.5 rounded-md text-xs font-medium transition-colors truncate"
              :class="importMode === m.id ? 'bg-blue-600 text-white' : 'text-gray-400 hover:text-white'"
            >{{ m.label }}</button>
          </div>

          <!-- Mode 1: Token JSON files (direct, no API call) -->
          <div v-if="importMode === 'token-files'">
            <div class="bg-green-900/20 border border-green-700/40 rounded-lg p-3 text-xs text-green-300 mb-3">
              <div class="flex items-start justify-between gap-2">
                <div>
                  ⚡ 直接解析 token 文件（无需调用 OpenAI API，速度最快）<br/>
                  支持单对象 JSON、数组 JSON、每行一个对象的 NDJSON，适合 <code class="text-green-200">token_*.json</code> 和 <code class="text-green-200">codex_tokens_*.json</code>
                </div>
                <button @click="downloadExample('token-files')" class="shrink-0 px-2 py-1 bg-green-800/60 hover:bg-green-700/80 text-green-200 rounded text-xs transition-colors whitespace-nowrap">
                  下载示例
                </button>
              </div>
            </div>
            <div v-if="!importFiles.length">
              <input ref="multiFileInput" type="file" accept=".json" multiple class="hidden" @change="handleMultiFileSelect"/>
              <div
                @click="$refs.multiFileInput.click()"
                @dragover.prevent="dragging = true"
                @dragleave.prevent="dragging = false"
                @drop.prevent="handleDrop"
                class="border-2 border-dashed border-gray-600 rounded-xl p-8 text-center cursor-pointer hover:border-blue-500 hover:bg-blue-900/10 transition-colors"
                :class="{ 'border-blue-500 bg-blue-900/10': dragging }"
              >
                <svg class="w-10 h-10 mx-auto mb-3 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"/>
                </svg>
                <p class="text-gray-400 text-sm">点击或拖拽文件到此处</p>
                <p class="text-xs text-gray-600 mt-1">支持多文件上传，也支持单个文件内包含多个账号</p>
              </div>
            </div>
            <div v-else>
              <div class="flex items-center justify-between mb-2">
                <span class="text-sm text-gray-300">已选择 <strong class="text-white">{{ importFiles.length }}</strong> 个文件</span>
                <button @click="importFiles = []; importResults = null" class="text-xs text-gray-500 hover:text-red-400">重新选择</button>
              </div>
              <div class="max-h-36 overflow-y-auto bg-gray-800 rounded-lg p-3 space-y-1">
                <div v-for="(f, i) in importFiles" :key="i" class="text-xs text-gray-400 truncate">
                  {{ i + 1 }}. {{ f.name }}
                </div>
              </div>
            </div>
          </div>

          <!-- Mode 2: 自适应导入（自动识别格式，单文件或多文件） -->
          <div v-if="importMode === 'auto-files'">
            <div class="bg-blue-900/20 border border-blue-700/40 rounded-lg p-3 text-xs text-blue-300 mb-3">
              <div class="flex items-start justify-between gap-2">
                <div>
                  🎯 选择 JSON 文件后<strong class="text-blue-200">自动识别</strong>格式并导入，无需知道文件属于哪种导出工具<br/>
                  <span class="text-blue-400/70">支持单个或多个文件；自动适配 Token、CPA、EasyLLM 备份；单文件内也支持数组、NDJSON</span>
                </div>
              </div>
            </div>
            <div v-if="!importAutoFiles.length">
              <input ref="importAutoFileInput" type="file" accept=".json,application/json" multiple class="hidden" @change="handleAutoFileSelect"/>
              <input ref="importAutoDirectoryInput" type="file" accept=".json,application/json" multiple webkitdirectory directory class="hidden" @change="handleAutoDirectorySelect"/>
              <div
                @dragover.prevent="dragging = true"
                @dragleave.prevent="dragging = false"
                @drop.prevent="handleDrop"
                class="border-2 border-dashed border-gray-600 rounded-xl p-8 text-center hover:border-blue-500 hover:bg-blue-900/10 transition-colors"
                :class="{ 'border-blue-500 bg-blue-900/10': dragging }"
              >
                <svg class="w-10 h-10 mx-auto mb-3 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"/>
                </svg>
                <p class="text-gray-400 text-sm">选择文件夹或 JSON 文件</p>
                <p class="text-xs text-gray-600 mt-1">文件夹内会递归导入 <code class="text-blue-300">.json</code>，格式混放也可以</p>
                <div class="mt-4 flex flex-col sm:flex-row items-center justify-center gap-2">
                  <button
                    @click.stop="openImportDirectoryPicker"
                    :disabled="selectingImportDirectory"
                    class="btn btn-primary text-xs px-3 py-1.5"
                  >
                    {{ selectingImportDirectory ? '打开中...' : '选择文件夹' }}
                  </button>
                  <button
                    @click.stop="$refs.importAutoFileInput.click()"
                    class="btn btn-secondary text-xs px-3 py-1.5"
                  >
                    选择 JSON 文件
                  </button>
                </div>
                <p class="text-xs text-gray-600 mt-3">也可以直接拖拽 JSON 文件到此处</p>
              </div>
            </div>
            <div v-else>
              <div class="flex items-center justify-between mb-2">
                <span class="text-sm text-gray-300">已选择 <strong class="text-white">{{ importAutoFiles.length }}</strong> 个文件</span>
                <button @click="resetScanImportSelection" class="text-xs text-gray-500 hover:text-red-400">重新选择</button>
              </div>
              <div class="max-h-36 overflow-y-auto bg-gray-800 rounded-lg p-3 space-y-1">
                <div v-for="(f, i) in importAutoFiles" :key="i" class="text-xs text-gray-400 truncate">
                  {{ i + 1 }}. {{ f.webkitRelativePath || f.name }}
                </div>
              </div>
            </div>
          </div>

          <!-- Mode 3: refresh_token list (legacy) -->
          <div v-if="importMode === 'refresh-tokens'">
            <div class="bg-yellow-900/20 border border-yellow-700/40 rounded-lg p-3 text-xs text-yellow-300 mb-3">
              <div class="flex items-start justify-between gap-2">
                <div>
                  🔄 通过 refresh_token 列表导入（需要调用 OpenAI API 获取账号信息，速度较慢）
                </div>
                <button @click="downloadExample('refresh-tokens')" class="shrink-0 px-2 py-1 bg-yellow-800/60 hover:bg-yellow-700/80 text-yellow-200 rounded text-xs transition-colors whitespace-nowrap">
                  下载示例
                </button>
              </div>
            </div>
            <div v-if="!importTokens.length">
              <input ref="fileInput" type="file" accept=".json,.txt" class="hidden" @change="handleFileSelect"/>
              <div
                @click="$refs.fileInput.click()"
                @dragover.prevent="dragging = true"
                @dragleave.prevent="dragging = false"
                @drop.prevent="handleDrop"
                class="border-2 border-dashed border-gray-600 rounded-xl p-8 text-center cursor-pointer hover:border-blue-500 hover:bg-blue-900/10 transition-colors"
                :class="{ 'border-blue-500 bg-blue-900/10': dragging }"
              >
                <svg class="w-10 h-10 mx-auto mb-3 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"/>
                </svg>
                <p class="text-gray-400 text-sm">点击或拖拽文件到此处</p>
                <pre class="text-xs text-gray-600 mt-2">["rt_xxx", "rt_yyy"]</pre>
              </div>
            </div>
            <div v-else>
              <div class="flex items-center justify-between mb-2">
                <span class="text-sm text-gray-300">已解析 <strong class="text-white">{{ importTokens.length }}</strong> 个 token</span>
                <button @click="importTokens = []; importResults = null" class="text-xs text-gray-500 hover:text-red-400">重新选择</button>
              </div>
              <div class="max-h-36 overflow-y-auto bg-gray-800 rounded-lg p-3 space-y-1">
                <div v-for="(t, i) in importTokens" :key="i" class="text-xs text-gray-400 font-mono truncate">
                  {{ i + 1 }}. {{ maskToken(t) }}
                </div>
              </div>
            </div>
          </div>

          <!-- Mode 4: CPA JSON（*-cpa.json / *.codex.cpa.json） -->
          <div v-if="importMode === 'cpa'">
            <div class="bg-violet-900/20 border border-violet-700/40 rounded-lg p-3 text-xs text-violet-300 mb-3">
              <div class="flex items-start justify-between gap-2">
                <div>
                  📋 导入 CPA 格式账号 JSON，无需调用 OpenAI API。<br/>
                  支持单文件单账号、单文件多账号数组，以及一次选择多个 <code class="text-violet-200">*-cpa.json</code> 文件。
                </div>
                <button @click="downloadExample('cpa')" class="shrink-0 px-2 py-1 bg-violet-800/60 hover:bg-violet-700/80 text-violet-200 rounded text-xs transition-colors whitespace-nowrap">
                  下载示例
                </button>
              </div>
            </div>
            <div v-if="!importCPAFiles.length">
              <input ref="importCPAFileInput" type="file" accept=".json,application/json" multiple class="hidden" @change="handleCPAFileSelect"/>
              <div
                @click="$refs.importCPAFileInput.click()"
                @dragover.prevent="dragging = true"
                @dragleave.prevent="dragging = false"
                @drop.prevent="handleDrop"
                class="border-2 border-dashed border-gray-600 rounded-xl p-8 text-center cursor-pointer hover:border-violet-500 hover:bg-violet-900/10 transition-colors"
                :class="{ 'border-violet-500 bg-violet-900/10': dragging }"
              >
                <svg class="w-10 h-10 mx-auto mb-3 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"/>
                </svg>
                <p class="text-gray-400 text-sm">点击或拖拽 CPA JSON 到此处</p>
                <p class="text-xs text-gray-600 mt-1">例如 <code class="text-violet-300">email-cpa.json</code>、<code class="text-violet-300">*.codex.cpa.json</code></p>
              </div>
            </div>
            <div v-else>
              <div class="flex items-center justify-between mb-2">
                <span class="text-sm text-gray-300">
                  已选择 <strong class="text-white">{{ importCPAFiles.length }}</strong> 个文件，
                  约 <strong class="text-white">{{ importCPAAccountCount }}</strong> 个账号
                </span>
                <button @click="importCPAFiles = []; importCPAAccountCount = 0; importResults = null" class="text-xs text-gray-500 hover:text-red-400">重新选择</button>
              </div>
              <div class="max-h-36 overflow-y-auto bg-gray-800 rounded-lg p-3 space-y-1">
                <div v-for="(item, i) in importCPAFiles" :key="i" class="text-xs text-gray-400 truncate">
                  {{ i + 1 }}. {{ item.name }}（{{ item.count }} 个账号）
                </div>
              </div>
            </div>
          </div>

          <!-- Mode 5: Re-import from exported backup JSON -->
          <div v-if="importMode === 'from-export'">
            <div class="bg-purple-900/20 border border-purple-700/40 rounded-lg p-3 text-xs text-purple-300 mb-3">
              <div class="flex items-start justify-between gap-2">
                <div>
                  📦 直接导入由「服务配置 → 导出账号」生成的备份文件（无需任何 OpenAI API 调用，速度最快）
                </div>
                <button @click="downloadExample('from-export')" class="shrink-0 px-2 py-1 bg-purple-800/60 hover:bg-purple-700/80 text-purple-200 rounded text-xs transition-colors whitespace-nowrap">
                  下载示例
                </button>
              </div>
            </div>
            <div v-if="!importBackupFile">
              <input ref="importBackupFileInput" type="file" accept=".json" class="hidden" @change="handleBackupFileSelect"/>
              <div
                @click="$refs.importBackupFileInput.click()"
                @dragover.prevent="dragging = true"
                @dragleave.prevent="dragging = false"
                @drop.prevent="handleDrop"
                class="border-2 border-dashed border-gray-600 rounded-xl p-8 text-center cursor-pointer hover:border-purple-500 hover:bg-purple-900/10 transition-colors"
                :class="{ 'border-purple-500 bg-purple-900/10': dragging }"
              >
                <svg class="w-10 h-10 mx-auto mb-3 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"/>
                </svg>
                <p class="text-gray-400 text-sm">点击或拖拽备份文件到此处</p>
                <p class="text-xs text-gray-600 mt-1">仅支持「导出账号」生成的文件</p>
              </div>
            </div>
            <div v-else class="space-y-2">
              <div class="flex items-center justify-between">
                <div class="text-sm text-gray-300">
                  已解析备份：
                  <span class="text-white font-medium">{{ importBackupFile.oauth_accounts?.length ?? 0 }}</span> 个 OAuth 账号，
                  <span class="text-white font-medium">{{ importBackupFile.api_accounts?.length ?? 0 }}</span> 个 API 账号
                  <span v-if="importBackupFile.local_access" class="text-emerald-300">，包含本地 API 服务配置</span>
                </div>
                <button @click="importBackupFile = null; importResults = null" class="text-xs text-gray-500 hover:text-red-400">重新选择</button>
              </div>
              <div v-if="importBackupFile.exported_at" class="text-xs text-gray-500">
                备份时间：{{ new Date(importBackupFile.exported_at).toLocaleString() }}
              </div>
            </div>
          </div>

          <!-- Import progress/results -->
          <div v-if="importing" class="flex items-center gap-3 text-sm text-blue-300">
            <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
            </svg>
            正在导入，请稍候...
          </div>
          <div v-if="importResults && !importing" class="space-y-2">
            <div class="flex items-center gap-4 text-sm font-medium">
              <span class="text-green-400">✓ 成功 {{ importResults.success }}</span>
              <span v-if="importResults.skipped" class="text-yellow-400">↷ 跳过 {{ importResults.skipped }}</span>
              <span class="text-red-400">✗ 失败 {{ importResults.failed }}</span>
              <span class="text-gray-500">共 {{ importResults.total }}</span>
            </div>
            <div class="max-h-52 overflow-y-auto bg-gray-800 rounded-lg p-3 space-y-1">
              <div v-for="r in importResults.results" :key="r.filename || r.index" class="flex items-start gap-2 text-xs py-0.5">
                <span class="shrink-0" :class="r.success ? 'text-green-400' : r.skipped ? 'text-yellow-400' : 'text-red-400'">
                  {{ r.success ? '✓' : r.skipped ? '↷' : '✗' }}
                </span>
                <span class="text-gray-300 truncate flex-1">{{ importResultDisplayLabel(r) }}</span>
                <span v-if="r.error && !r.skipped" class="text-red-400 shrink-0 truncate max-w-[160px]">{{ r.error }}</span>
              </div>
            </div>
          </div>
        </div>

        <div class="flex flex-wrap justify-end gap-3 p-6 border-t border-gray-700">
          <button @click="closeImportDialog" class="btn btn-secondary" :disabled="importing">关闭</button>
          <button
            v-if="canRunImport && !importResults"
            @click="runImport"
            :disabled="importing"
            class="btn btn-primary"
          >
            {{ importing ? '导入中...' : importButtonLabel }}
          </button>
        </div>
      </div>
    </div>

    <!-- OAuth Login Dialog -->
    <div v-if="showOAuthDialog" class="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4">
      <div class="bg-gray-900 border border-gray-700 rounded-2xl w-full max-w-md shadow-2xl">
        <div class="flex items-center justify-between p-6 border-b border-gray-700">
          <h2 class="text-lg font-semibold text-white">OpenAI OAuth 登录</h2>
          <button @click="closeOAuthDialog" class="text-gray-400 hover:text-white">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>
        <div class="p-6 space-y-4">
          <div v-if="!oauthState.authUrl">
            <p class="text-gray-400 text-sm mb-4">点击下方按钮后会自动打开浏览器，并等待本机回调自动完成登录；如果自动回调不可用，也可以手动粘贴完整回调地址或 `code`。</p>
            <button @click="generateOAuthUrl" :disabled="oauthState.loading" class="btn btn-primary w-full">
              {{ oauthState.loading ? '准备中...' : '生成授权链接并打开浏览器' }}
            </button>
          </div>
          <div v-else class="space-y-4">
            <div v-if="oauthState.autoCallbackEnabled" class="rounded-xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-200">
              授权页已准备好。完成 OpenAI 登录后，这里会自动继续，无需手动复制 `code`。
            </div>
            <div v-else class="rounded-xl border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-100">
              当前未启用本地自动回调，请在授权后手动粘贴完整回调地址或 `authorization_code`。
            </div>
            <div v-if="oauthState.autoCallbackError" class="rounded-xl border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-100">
              {{ oauthState.autoCallbackError }}
            </div>
            <div v-if="oauthState.status === 'callback_received' && oauthState.loading" class="rounded-xl border border-blue-500/30 bg-blue-500/10 px-4 py-3 text-sm text-blue-100">
              已收到本地回调，正在完成登录...
            </div>
            <div>
              <label class="block text-xs text-gray-400 mb-1">授权链接（在浏览器中打开）</label>
              <div class="flex gap-2">
                <input readonly :value="oauthState.authUrl" class="input flex-1 text-xs font-mono"/>
                <button @click="openOAuthInBrowser" class="btn btn-secondary text-xs px-3">打开</button>
                <button @click="copyAuthUrl" class="btn btn-secondary text-xs px-3">复制</button>
              </div>
            </div>
            <div>
              <label class="block text-xs text-gray-400 mb-1">如果没有自动返回，粘贴完整回调地址或 `authorization_code`</label>
              <input
                v-model="oauthState.manualInput"
                class="input w-full"
                placeholder="例如：http://localhost:1455/auth/callback?code=...&state=... 或直接粘贴 code"
              />
            </div>
            <button
              @click="exchangeOAuthCode"
              :disabled="!oauthState.sessionId || oauthState.loading"
              class="btn btn-primary w-full"
            >
              {{ oauthState.loading ? '验证中...' : '我已授权，继续登录' }}
            </button>
          </div>
          <p v-if="oauthState.error" class="text-red-400 text-sm">{{ oauthState.error }}</p>
        </div>
      </div>
    </div>

    <!-- Add/Edit API Account Dialog -->
    <div v-if="showAddAPIDialog" class="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4">
      <div class="bg-gray-900 border border-gray-700 rounded-2xl w-full max-w-lg shadow-2xl">
        <div class="flex items-center justify-between p-6 border-b border-gray-700">
          <h2 class="text-lg font-semibold text-white">{{ editingAPIAccount ? '编辑' : '添加' }} API 账号</h2>
          <button @click="closeAPIDialog" class="text-gray-400 hover:text-white">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>
        <div class="p-6 space-y-4">
          <div>
            <label class="block text-xs text-gray-400 mb-1">Model Provider <span class="text-red-400">*</span></label>
            <input v-model="apiForm.model_provider" class="input w-full" placeholder="openai"/>
          </div>
          <div>
            <label class="block text-xs text-gray-400 mb-1">Model <span class="text-red-400">*</span></label>
            <input v-model="apiForm.model" class="input w-full" placeholder="e.g. gpt-4o"/>
          </div>
          <div>
            <label class="block text-xs text-gray-400 mb-1">Base URL <span class="text-red-400">*</span></label>
            <input v-model="apiForm.base_url" class="input w-full" placeholder="https://api.openai.com/v1"/>
          </div>
          <div>
            <label class="block text-xs text-gray-400 mb-1">API Key <span class="text-red-400">*</span></label>
            <input v-model="apiForm.api_key" class="input w-full" type="password" placeholder="sk-..."/>
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-xs text-gray-400 mb-1">Wire API</label>
              <select v-model="apiForm.wire_api" class="input w-full">
                <option value="responses">responses</option>
                <option value="chat">chat</option>
              </select>
            </div>
            <div>
              <label class="block text-xs text-gray-400 mb-1">Reasoning Effort</label>
              <select v-model="apiForm.model_reasoning_effort" class="input w-full">
                <option value="">不设置</option>
                <option value="low">low</option>
                <option value="medium">medium</option>
                <option value="high">high</option>
                <option value="xhigh">xhigh</option>
              </select>
            </div>
          </div>
          <p v-if="apiFormError" class="text-red-400 text-sm">{{ apiFormError }}</p>
        </div>
        <div class="flex justify-end gap-3 p-6 border-t border-gray-700">
          <button @click="closeAPIDialog" class="btn btn-secondary">取消</button>
          <button @click="saveAPIAccount" :disabled="savingAPI" class="btn btn-primary">
            {{ savingAPI ? '保存中...' : '保存' }}
          </button>
        </div>
      </div>
    </div>
    <!-- Service Config Dialog -->
    <div v-if="showServiceConfigDialog" class="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-6">
      <div class="bg-gray-900 border border-gray-700 rounded-2xl w-full max-w-[calc(100vw-3rem)] xl:max-w-7xl shadow-2xl">
        <div class="flex items-center justify-between p-6 border-b border-gray-700">
          <h2 class="text-lg font-semibold text-white flex items-center gap-2">
            <svg class="w-5 h-5 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
            </svg>
            服务配置
          </h2>
          <div class="flex items-center gap-2">
            <button
              @click="exportAccounts"
              :disabled="exportingAccounts"
              class="flex items-center gap-1.5 px-3 py-1.5 bg-gray-700 hover:bg-gray-600 text-gray-200 rounded-lg text-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/>
              </svg>
              {{ exportingAccounts ? '导出中...' : '导出账号' }}
            </button>
            <button @click="showServiceConfigDialog = false" class="text-gray-400 hover:text-white">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
          </div>
        </div>
        <div class="p-6 space-y-5 max-h-[80vh] overflow-y-auto">



          <!-- Stats Cards -->
          <div class="grid grid-cols-3 gap-3">
            <div class="bg-gray-800 rounded-xl p-4 text-center">
              <div class="text-2xl font-bold text-blue-400">{{ serviceConfig.pool_size }}</div>
              <div class="text-xs text-gray-400 mt-1">池中账号</div>
            </div>
            <div class="bg-gray-800 rounded-xl p-4 text-center">
              <div class="text-2xl font-bold text-green-400">{{ serviceConfig.total_requests }}</div>
              <div class="text-xs text-gray-400 mt-1">转发请求数</div>
            </div>
            <div class="bg-gray-800 rounded-xl p-4 text-center">
              <div class="text-2xl font-bold text-purple-400">不保留</div>
              <div class="text-xs text-gray-400 mt-1">调用日志</div>
            </div>
          </div>

          <!-- Codex API Service -->
          <div class="bg-gray-800 rounded-xl p-4 space-y-3">
            <div class="flex items-start justify-between gap-4">
              <div class="min-w-0">
                <div class="flex items-center gap-2">
                  <div class="text-sm font-medium text-white">Codex API 服务</div>
                  <span
                    class="text-[10px] px-2 py-0.5 rounded-full font-medium"
                    :class="serviceConfig.codex_api_service ? 'bg-green-500/20 text-green-300' : 'bg-gray-700 text-gray-500'"
                  >
                    {{ serviceConfig.codex_api_service ? '已注入' : '未注入' }}
                  </span>
                </div>
                <div class="text-xs text-gray-400 mt-0.5">
                  启动后自动写入本机 <code class="text-blue-300">~/.codex/auth.json</code> 和 <code class="text-blue-300">config.toml</code>，Codex 直接走 EasyLLM 本地服务。
                </div>
              </div>
              <button
                @click="activateCodexAPIService"
                :disabled="savingServiceConfig || oauthAccounts.length === 0"
                class="btn btn-sm btn-primary shrink-0"
                :title="oauthAccounts.length === 0 ? '请先导入 OAuth 账号' : '开启代理池并注入本机 Codex 配置'"
              >
                {{ savingServiceConfig ? '处理中...' : '启动并注入 Codex' }}
              </button>
            </div>
            <div class="grid md:grid-cols-2 gap-2 text-xs">
              <div class="flex items-center justify-between bg-gray-900/60 rounded-lg px-3 py-2 min-w-0">
                <span class="text-gray-500 shrink-0">Base URL</span>
                <code class="text-blue-300 font-mono truncate mx-3">{{ serviceConfig.codex_api_base_url || serviceAPIBaseURL }}</code>
                <button @click="copyText(serviceConfig.codex_api_base_url || serviceAPIBaseURL)" class="text-gray-500 hover:text-white shrink-0">复制</button>
              </div>
              <div class="flex items-center justify-between bg-gray-900/60 rounded-lg px-3 py-2 min-w-0">
                <span class="text-gray-500 shrink-0">Wire API</span>
                <code class="text-emerald-300 font-mono truncate mx-3">responses</code>
                <span class="text-gray-600 shrink-0">model_provider=easyllm</span>
              </div>
            </div>
          </div>

          <!-- Codex local API service -->
          <div class="bg-gray-800 rounded-xl p-4 space-y-4">
            <div class="flex items-start justify-between gap-4">
              <div>
                <div class="text-sm font-medium text-white">Codex 本地 API 服务</div>
                <div class="text-xs text-gray-400 mt-0.5">
                  管理注入到本机 Codex 的账号集合、端口和调度策略。
                </div>
              </div>
              <div class="flex items-center gap-2 shrink-0">
                <span class="text-[10px] px-2 py-0.5 rounded-full font-medium"
                  :class="localAccess.running ? 'bg-green-500/20 text-green-300' : 'bg-gray-700 text-gray-500'">
                  {{ localAccess.running ? '运行中' : '已停止' }}
                </span>
                <button @click="activateLocalAccess" :disabled="localAccessBusy || oauthAccounts.length === 0" class="btn btn-sm btn-primary">
                  {{ localAccessBusy ? '处理中...' : '启动/注入' }}
                </button>
                <button @click="deactivateLocalAccess" :disabled="localAccessBusy || !localAccess.collection?.enabled" class="btn btn-sm btn-secondary">
                  停止
                </button>
              </div>
            </div>

            <div class="grid md:grid-cols-3 gap-2 text-xs">
              <div class="bg-gray-900/60 rounded-lg px-3 py-2 min-w-0">
                <div class="text-gray-500">成员账号</div>
                <div class="mt-1 text-lg font-semibold text-white">{{ localAccess.member_count || 0 }}</div>
              </div>
              <div class="bg-gray-900/60 rounded-lg px-3 py-2 min-w-0">
                <div class="text-gray-500">API Key</div>
                <code class="mt-1 block text-green-300 truncate">{{ localAccess.collection?.api_key_masked || '未设置' }}</code>
              </div>
              <div class="bg-gray-900/60 rounded-lg px-3 py-2 min-w-0">
                <div class="text-gray-500">入口</div>
                <div class="mt-1 flex items-center gap-2 min-w-0">
                  <code class="text-blue-300 truncate">{{ localAccess.api_port_url || serviceConfig.codex_api_port_url || '' }}</code>
                  <button @click="copyText(localAccess.api_port_url || serviceConfig.codex_api_port_url || '')" class="text-gray-500 hover:text-white shrink-0">复制</button>
                </div>
              </div>
            </div>

            <div class="grid lg:grid-cols-[1fr_1fr] gap-4">
              <div class="space-y-3">
                <div class="grid gap-1">
                  <input :value="localAccessPortInput" class="input text-xs" readonly />
                  <div class="text-[11px] text-gray-500">端口跟随当前 EasyLLM 服务启动配置</div>
                </div>
                <div class="grid sm:grid-cols-[1fr_auto] gap-2">
                  <select
                    :value="localAccess.collection?.routing_strategy || serviceConfig.strategy"
                    @change="updateLocalAccessRouting($event.target.value)"
                    class="input text-xs"
                  >
                    <option v-for="s in strategies" :key="s.id" :value="s.id">{{ s.label }}</option>
                  </select>
                  <button @click="rotateLocalAccessKey" :disabled="localAccessBusy" class="btn btn-sm btn-secondary">重置 Key</button>
                </div>
                <label class="flex items-center gap-2 text-xs text-gray-300">
                  <input v-model="localAccessRestrictFree" type="checkbox" class="h-4 w-4 rounded border-gray-600 bg-gray-900 text-blue-500" />
                  <span>保存集合时排除免费账号</span>
                </label>
              </div>

              <div class="space-y-2">
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <div class="text-xs font-medium text-gray-300">API 服务账号集合</div>
                  <div class="flex flex-wrap items-center gap-2">
                    <span class="text-[11px] text-gray-500">{{ localAccessSelectedCount }}/{{ localAccessEligibleCount }}</span>
                    <button @click="selectAllLocalAccessAccounts" :disabled="localAccessBusy || localAccessEligibleCount === 0" class="btn btn-xs btn-secondary">全选成功</button>
                    <button @click="clearLocalAccessAccounts" :disabled="localAccessBusy || localAccessSelectedCount === 0" class="btn btn-xs btn-secondary">清空</button>
                    <button @click="saveAllLocalAccessAccounts" :disabled="localAccessBusy || localAccessEligibleCount === 0" class="btn btn-xs btn-primary">加入 200</button>
                    <button @click="saveLocalAccessAccounts" :disabled="localAccessBusy" class="btn btn-xs btn-primary">保存集合</button>
                  </div>
                </div>
                <div class="max-h-36 overflow-y-auto rounded-lg border border-gray-700 bg-gray-900/60 p-2 space-y-1">
                  <label v-for="account in oauthAccounts" :key="account.id" class="flex items-center gap-2 rounded px-2 py-1 text-xs text-gray-300 hover:bg-gray-800" :class="{ 'opacity-50': !isLocalAccessEligibleAccount(account) }">
                    <input
                      type="checkbox"
                      :checked="localAccessSelectedIds.includes(accountId(account.id)) && isLocalAccessEligibleAccount(account)"
                      :disabled="!isLocalAccessEligibleAccount(account)"
                      @change="toggleLocalAccessAccount(account.id)"
                    />
                    <span class="truncate flex-1" :title="accountDisplayTitle(account)">{{ accountDisplayLabel(account) }}</span>
                    <span v-if="account.plan" class="text-gray-500">{{ account.plan }}</span>
                    <span v-if="Number(account._quota_http_status) === 200 && !account.quota_is_forbidden" class="text-green-400">200</span>
                    <span v-else class="text-gray-600">{{ account._quota_http_status || '未查' }}</span>
                  </label>
                  <div v-if="oauthAccounts.length === 0" class="text-xs text-gray-500 px-2 py-3 text-center">暂无 OAuth 账号</div>
                </div>
              </div>
            </div>
          </div>

          <!-- Proxy Pool Toggle -->
          <div class="flex items-center justify-between bg-gray-800 rounded-xl p-4">
            <div>
              <div class="text-sm font-medium text-white">代理池服务</div>
              <div class="text-xs text-gray-400 mt-0.5">控制 <code class="text-blue-300">/v1/*</code> 接口是否对外可用</div>
            </div>
            <button
              @click="toggleServiceProxyPool"
              :disabled="savingServiceConfig"
              class="relative w-12 h-6 rounded-full transition-colors duration-200 focus:outline-none"
              :class="serviceConfig.proxy_pool_enabled ? 'bg-green-500' : 'bg-gray-600'"
            >
              <span class="absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full shadow transition-transform duration-200"
                :class="serviceConfig.proxy_pool_enabled ? 'translate-x-6' : 'translate-x-0'"></span>
            </button>
          </div>

          <!-- Proxy Pool Batch Toggle -->
          <div class="bg-gray-800 rounded-xl p-4">
            <div class="flex items-center justify-between">
              <div>
                <div class="text-sm font-medium text-white">轮询代理池</div>
                <div class="text-xs text-gray-400 mt-0.5"><code class="text-blue-300">/v1/responses</code> 请求在已加入的账号间轮询</div>
              </div>
              <div class="flex items-center gap-3 shrink-0">
                <span class="text-xs px-2 py-0.5 rounded-full font-medium"
                  :class="proxyEnabledCount > 0 ? 'bg-green-500/20 text-green-400' : 'bg-gray-700 text-gray-500'">
                  {{ proxyEnabledCount > 0 ? `${proxyEnabledCount} 个账号` : '无账号' }}
                </span>
                <button
                  type="button"
                  @click="toggleProxyAll(!proxyAllOn)"
                  :disabled="togglingProxyAll || oauthAccounts.length === 0"
                  class="flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-medium transition-all shrink-0"
                  :class="proxyAllOn
                    ? 'bg-green-500/25 border border-green-500/50 text-green-300 hover:bg-green-500/35'
                    : 'bg-gray-700/80 border border-gray-600 text-gray-300 hover:bg-gray-600'"
                  :title="proxyAllOn ? '一键移出：将所有 OAuth 账号移出代理池' : '一键加入：将所有 OAuth 账号加入代理池'"
                >
                  <span class="inline-block w-2 h-2 rounded-full" :class="proxyAllOn ? 'bg-green-400' : 'bg-gray-500'"></span>
                  <span v-if="togglingProxyAll">处理中...</span>
                  <span v-else>{{ proxyAllOn ? '一键全部移出' : '一键全部加入' }}</span>
                </button>
              </div>
            </div>
          </div>

          <!-- Proxy Endpoints -->
          <div class="bg-gray-800 rounded-xl p-4 space-y-3">
            <div class="flex items-center gap-2">
              <svg class="w-4 h-4 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/>
              </svg>
              <div class="text-sm font-medium text-white">接入端点</div>
              <span class="text-xs text-gray-500">在 IDE / 工具中使用以下地址</span>
            </div>
            <div class="space-y-2">
              <div v-for="ep in proxyEndpoints" :key="ep.method + ep.path"
                class="flex items-center justify-between bg-gray-900/60 rounded-lg px-3 py-2 group">
                <div class="flex items-center gap-3 min-w-0">
                  <span class="shrink-0 text-xs font-bold px-1.5 py-0.5 rounded font-mono"
                    :class="ep.method === 'GET' ? 'bg-green-500/20 text-green-400' : 'bg-orange-500/20 text-orange-400'">
                    {{ ep.method }}
                  </span>
                  <code class="text-blue-300 text-xs font-mono truncate">{{ baseURL + ep.path }}</code>
                  <span class="text-gray-500 text-xs shrink-0 hidden group-hover:inline">{{ ep.desc }}</span>
                </div>
                <button @click="copyText(baseURL + ep.path)" class="shrink-0 ml-3 text-xs text-gray-500 hover:text-white bg-gray-700 hover:bg-gray-600 px-2 py-1 rounded transition-colors">
                  复制
                </button>
              </div>
            </div>
          </div>

          <!-- Strategy -->
          <div class="bg-gray-800 rounded-xl p-4">
            <div class="text-sm font-medium text-white mb-2">轮询策略</div>
            <div class="flex gap-2">
              <button v-for="s in strategies" :key="s.id"
                @click="updateServiceStrategy(s.id)"
                :disabled="savingServiceConfig"
                class="flex-1 py-2 px-3 rounded-lg text-xs font-medium transition-all border"
                :class="serviceConfig.strategy === s.id
                  ? 'bg-blue-600/20 border-blue-500/50 text-blue-300'
                  : 'bg-gray-700/50 border-gray-600 text-gray-400 hover:text-white hover:border-gray-500'"
              >{{ s.label }}</button>
            </div>
          </div>

          <!-- API Key -->
          <div class="bg-gray-800 rounded-xl p-4 space-y-3">
            <div>
              <div class="text-sm font-medium text-white">对外 API Key</div>
              <div class="text-xs text-gray-400 mt-0.5">设置后，外部调用 <code class="text-blue-300">/v1/responses</code> 需在 Header 携带 <code class="text-blue-300">Authorization: Bearer &lt;key&gt;</code></div>
            </div>
            <div class="flex gap-2">
              <input
                v-model="serviceApiKeyInput"
                class="input flex-1 font-mono text-xs"
                :type="showApiKey ? 'text' : 'password'"
                placeholder="留空则不鉴权（任何人可调用）"
              />
              <button @click="showApiKey = !showApiKey" class="btn btn-sm btn-ghost shrink-0" :title="showApiKey ? '隐藏' : '显示'">
                <svg v-if="showApiKey" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/>
                </svg>
                <svg v-else class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21"/>
                </svg>
              </button>
            </div>
            <div class="flex items-center justify-between">
              <span v-if="serviceConfig.api_key_set" class="text-xs text-green-400 flex items-center gap-1.5">
                当前已设置:
                <code class="text-green-300 bg-green-500/10 px-1.5 py-0.5 rounded">{{ serviceConfig.api_key || serviceConfig.api_key_masked || '****' }}</code>
              </span>
              <span v-else class="text-xs text-yellow-400">
                未设置（无需鉴权即可调用）
              </span>
              <button
                @click="saveServiceApiKey"
                :disabled="savingServiceConfig"
                class="btn btn-sm btn-primary"
              >{{ savingServiceConfig ? '保存中...' : '保存 Key' }}</button>
            </div>
          </div>

        </div>
      </div>
    </div>

  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount, inject, watch } from 'vue'
import { useRoute } from 'vue-router'
import api, { longApi, openaiAPI } from '@/api/index.js'
import CodexIcon from '@/components/CodexIcon.vue'
import { getCodexRouteMeta } from '@/lib/codexRoutes'
import { filterAPIAccounts, filterOAuthAccounts } from '@/lib/accounts'
import { localServiceAPIBaseURL, localServiceOrigin } from '@/lib/runtime'

// State
const route = useRoute()
const notify = inject('notify')
const confirmOperation = inject('confirmOperation')
const currentCodexRoute = computed(() => getCodexRouteMeta('codex'))
const pageTitle = computed(() => 'OpenAI / Codex 管理')
const pageSubtitle = computed(() => '管理 OpenAI OAuth 账号及 Codex API 配置')
const accounts = ref([])
const loading = ref(false)
const activeTab = ref('oauth')

// Proxy endpoints
const baseURL = computed(() => localServiceOrigin())
const serviceAPIBaseURL = computed(() => localServiceAPIBaseURL())
const proxyEndpoints = [
  { method: 'POST', path: '/v1/responses', desc: 'Codex Responses（推荐）' },
  { method: 'POST', path: '/v1/chat/completions', desc: '聊天补全（兼容）' },
  { method: 'GET',  path: '/v1/models',           desc: '获取可用模型列表' },
  { method: 'GET',  path: '/pool/status',         desc: '查看代理账号池状态' },
]

function copyText(text) {
  navigator.clipboard.writeText(text).then(() => showToast('已复制', 'success'))
}
const switchingId = ref(null)
const testingAPIId = ref(null)
const refreshingId = ref(null)
const refreshingAllTokens = ref(false)
const togglingProxyId = ref(null)
const togglingProxyAll = ref(false)
const fetchingQuotas = ref(false)
const fetchingQuotaIds = ref([])
const quotaLastFetched = ref('')
const planGroupFilter = ref('all')
const quotaFilter = ref('all') // all | 200 | 401 | 403 | 429 | 503
const searchQuery = ref('') // search by email
const apiSearchQuery = ref('')
const bulkDeleting = ref(false)
const showBulkDeleteConfirm = ref(false)
const bulkDeleteIds = ref([])
const bulkDeletePreview = ref([])
const bulkDeleteSelectionType = ref('oauth')
const bulkDeleteAccountLabel = ref('OAuth 账号')
const bulkDeleteScopeLabel = ref('')
const bulkDeleteAllSelected = ref(false)
const selectedOAuthIds = ref([])
const selectedAPIIds = ref([])
const hideAccountEmails = ref(readStoredBoolean('easyllm.openai.hideAccountEmails', false))
const accountLayoutModes = ['standard', 'dense']
const accountLayout = ref(readStoredOption('easyllm.openai.accountLayout', 'standard', accountLayoutModes))
const accountSortOptions = [
  { id: 'smart', label: '智能排序' },
  { id: 'updated_desc', label: '最近更新' },
  { id: 'quota_5h_desc', label: '5h 剩余高' },
  { id: 'quota_5h_asc', label: '5h 剩余低' },
  { id: 'plan_desc', label: '订阅优先' },
  { id: 'plan_asc', label: 'free 优先' },
  { id: 'email_asc', label: '邮箱 A-Z' },
]
const accountSortModeValues = accountSortOptions.map(option => option.id)
const accountSortMode = ref(readStoredOption('easyllm.openai.accountSortMode', 'smart', accountSortModeValues))
const storedAccountGroups = readStoredJSON('easyllm.openai.accountGroups', [])
const accountGroups = ref(Array.isArray(storedAccountGroups) ? storedAccountGroups.map(normalizeAccountGroup).filter(Boolean) : [])
const activeGroupFilter = ref(readStoredOption('easyllm.openai.activeGroupFilter', 'all', ['all', ...accountGroups.value.map(g => g.id)]))
const showGroupManager = ref(false)
const newGroupName = ref('')

const quotaFilterLabel = computed(() => {
  if (quotaFilter.value === '503') return '503（服务不可用）'
  if (quotaFilter.value === '429') return '429（限流）'
  if (quotaFilter.value === '403') return '403（地区受限/禁止）'
  if (quotaFilter.value === '401') return '401（失效/未授权）'
  if (quotaFilter.value === '200') return '200（成功）'
  return '全部'
})

// Import dialog
const showImportDialog = ref(false)
const importing = ref(false)
const importTokens = ref([])
const importFiles = ref([])
const importResults = ref(null)
const fileInput = ref(null)
const multiFileInput = ref(null)
const dragging = ref(false)
const importAutoFiles = ref([])
const importAutoFileInput = ref(null)
const importAutoDirectoryInput = ref(null)
const selectingImportDirectory = ref(false)
const importCPAFiles = ref([])
const importCPAFileInput = ref(null)
const importCPAAccountCount = ref(0)
const importMode = ref('token-files')
const importBackupFile = ref(null)  // 从备份导入用的解析后的 JSON 对象
const importBackupFileInput = ref(null)
const importModes = [
  { id: 'token-files',  label: '⚡ Token文件' },
  { id: 'auto-files',   label: '🎯 自适应' },
  { id: 'refresh-tokens', label: '🔄 refresh_token' },
  { id: 'cpa',          label: '📋 CPA' },
  { id: 'from-export',  label: '📦 从备份导入' },
]

// OAuth dialog
const showOAuthDialog = ref(false)
const OAUTH_POLL_INTERVAL_MS = 1500
let oauthPollTimer = null
const oauthState = ref(createEmptyOAuthState())

// API account dialog
const showAddAPIDialog = ref(false)
const editingAPIAccount = ref(null)
const savingAPI = ref(false)
const apiFormError = ref('')
const apiForm = ref({
  model_provider: '',
  model: '',
  base_url: '',
  api_key: '',
  wire_api: 'responses',
  model_reasoning_effort: ''
})
const providerDisplayNames = {
  openai: 'OpenAI'
}

const showServiceConfigDialog = ref(false)
const savingServiceConfig = ref(false)
const exportingAccounts = ref(false)
const showApiKey = ref(false)
const serviceApiKeyInput = ref('')
const serviceConfig = ref({
  proxy_pool_enabled: true,
  strategy: 'round_robin',
  pool_size: 0,
  proxy_enabled_count: 0,
  total_requests: 0,
  request_logs_retained: false,
  api_key_set: false,
  api_key: '',
  api_key_masked: '',
  v1_proxy_mode: '',
  codex_api_service: false,
  codex_api_base_url: '',
  codex_api_port_url: ''
})
const localAccess = ref({
  collection: {
    enabled: false,
    port: 8022,
    api_key_masked: '',
    routing_strategy: 'auto',
    restrict_free_accounts: true,
    account_ids: []
  },
  running: false,
  base_url: '',
  api_port_url: '',
  member_count: 0,
  stats: {
    daily: { totals: {}, accounts: [] },
    weekly: { totals: {}, accounts: [] },
    monthly: { totals: {}, accounts: [] }
  }
})
const localAccessBusy = ref(false)
const localAccessSelectedIds = ref([])
const localAccessPortInput = ref('')
const localAccessRestrictFree = ref(true)
const strategies = [
  { id: 'auto', label: '自动' },
  { id: 'quota_high_first', label: '优先高配额' },
  { id: 'quota_low_first', label: '优先低配额' },
  { id: 'plan_high_first', label: '优先高订阅' },
  { id: 'plan_low_first', label: '优先低订阅' },
  { id: 'expiry_soon_first', label: '优先近到期' },
  { id: 'round_robin', label: '轮询' },
  { id: 'random', label: '随机' },
  { id: 'least_used', label: '最少使用' }
]

// Delete confirm dialog
const showDeleteConfirm = ref(false)
const deleteTargetId = ref(null)
const deleteTargetLabel = ref('')
const deletingAccount = ref(false)

const formatExample = `[
  "refresh_token_1_here",
  "refresh_token_2_here",
  "refresh_token_3_here"
]`

// Computed
const oauthAccounts = computed(() => filterOAuthAccounts(accounts.value))
const apiAccounts = computed(() => filterAPIAccounts(accounts.value))
const proxyEnabledCount = computed(() => oauthAccounts.value.filter(a => a.proxy_enabled).length)
const proxyAllOn = computed(() => oauthAccounts.value.length > 0 && proxyEnabledCount.value === oauthAccounts.value.length)
const planGroups = computed(() => [
  { id: 'all', label: '账号类型', count: oauthAccounts.value.length },
  { id: 'team', label: 'team', count: countOAuthAccountsByPlan('team') },
  { id: 'plus', label: 'plus', count: countOAuthAccountsByPlan('plus') },
  { id: 'free', label: 'free', count: countOAuthAccountsByPlan('free') },
])
const activeGroup = computed(() => accountGroups.value.find(g => g.id === activeGroupFilter.value) || null)
const activeGroupAccountIDs = computed(() => new Set(activeGroup.value?.account_ids || []))
const localAccessEligibleAccounts = computed(() => oauthAccounts.value.filter(isLocalAccessEligibleAccount))
const localAccessEligibleIDSet = computed(() => new Set(localAccessEligibleAccounts.value.map(account => accountId(account.id))))
const localAccessSelectedCount = computed(() => localAccessSelectedIds.value.filter(id => localAccessEligibleIDSet.value.has(id)).length)
const localAccessEligibleCount = computed(() => localAccessEligibleAccounts.value.length)

const filteredOAuthAccounts = computed(() => {
  let list = oauthAccounts.value
  if (planGroupFilter.value !== 'all') {
    list = list.filter(a => accountPlanType(a) === planGroupFilter.value)
  }
  // Search filter
  const q = searchQuery.value.trim().toLowerCase()
  if (q) {
    list = list.filter(a => (a.email || '').toLowerCase().includes(q) || (a.chatgpt_account_id || '').toLowerCase().includes(q))
  }
  // Quota status filter
  const f = quotaFilter.value
  if (f !== 'all') {
    const want = Number(f)
    list = list.filter(a => {
      if (Number(a._quota_http_status) === want) return true
      if (want === 401 && a.status === 'reauth_required') return true
      return false
    })
  }
  return sortOAuthAccounts(list)
})
const filteredAPIAccounts = computed(() => {
  let list = apiAccounts.value
  const q = apiSearchQuery.value.trim().toLowerCase()
  if (q) {
    list = list.filter(a => [
      a.email,
      a.model_provider,
      a.model,
      a.base_url,
      a.wire_api,
    ].some(value => String(value || '').toLowerCase().includes(q)))
  }
  return sortAPIAccounts(list)
})

const allFilteredOAuthSelected = computed(() => (
  filteredOAuthAccounts.value.length > 0 &&
  filteredOAuthAccounts.value.every(a => selectedOAuthIds.value.includes(accountId(a.id)))
))
const allAPISelected = computed(() => (
  filteredAPIAccounts.value.length > 0 &&
  filteredAPIAccounts.value.every(a => selectedAPIIds.value.includes(accountId(a.id)))
))
const accountGridClass = computed(() => accountLayout.value === 'dense'
  ? 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5 2xl:grid-cols-6 gap-2'
  : 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-3 2xl:grid-cols-4 gap-3'
)

// Pagination
const PAGE_SIZE = 20
const DENSE_PAGE_SIZE = 48
const MAX_PAGE_SIZE = 2000
const pageSizeOptions = [12, 20, 32, 48, 50, 96, 100, 200, 500, 1000, 2000]
const defaultPageSize = accountLayout.value === 'dense' ? DENSE_PAGE_SIZE : PAGE_SIZE
const accountPageSize = ref(normalizeAccountPageSize(readStoredAccountPageSize(defaultPageSize)))
const currentPageSize = computed(() => normalizeAccountPageSize(accountPageSize.value))
const oauthPage = ref(1)
const apiPage = ref(1)
const oauthTotalPages = computed(() => Math.ceil(filteredOAuthAccounts.value.length / currentPageSize.value) || 1)
const apiTotalPages = computed(() => Math.ceil(filteredAPIAccounts.value.length / currentPageSize.value) || 1)
const oauthPaginationRangeText = computed(() => paginationRangeText(oauthPage.value, filteredOAuthAccounts.value.length, currentPageSize.value))
const apiPaginationRangeText = computed(() => paginationRangeText(apiPage.value, filteredAPIAccounts.value.length, currentPageSize.value))
const paginatedOAuth = computed(() => {
  const start = (oauthPage.value - 1) * currentPageSize.value
  return filteredOAuthAccounts.value.slice(start, start + currentPageSize.value)
})
const paginatedAPI = computed(() => {
  const start = (apiPage.value - 1) * currentPageSize.value
  return filteredAPIAccounts.value.slice(start, start + currentPageSize.value)
})

const tabs = computed(() => [
  { id: 'oauth', label: 'OAuth 账号', count: oauthAccounts.value.length },
  { id: 'api', label: 'API 账号', count: apiAccounts.value.length }
])

function readStoredOption(key, fallback, allowedValues) {
  try {
    const value = localStorage.getItem(key)
    return allowedValues.includes(value) ? value : fallback
  } catch {
    return fallback
  }
}

function writeStoredOption(key, value) {
  try {
    localStorage.setItem(key, value)
  } catch {}
}

function readStoredBoolean(key, fallback = false) {
  try {
    const value = localStorage.getItem(key)
    if (value === 'true') return true
    if (value === 'false') return false
    return fallback
  } catch {
    return fallback
  }
}

function writeStoredBoolean(key, value) {
  try {
    localStorage.setItem(key, value ? 'true' : 'false')
  } catch {}
}

function readStoredJSON(key, fallback) {
  try {
    const raw = localStorage.getItem(key)
    if (!raw) return fallback
    const parsed = JSON.parse(raw)
    return parsed == null ? fallback : parsed
  } catch {
    return fallback
  }
}

function writeStoredJSON(key, value) {
  try {
    localStorage.setItem(key, JSON.stringify(value))
  } catch {}
}

function toggleAccountPrivacy() {
  hideAccountEmails.value = !hideAccountEmails.value
  writeStoredBoolean('easyllm.openai.hideAccountEmails', hideAccountEmails.value)
  showToast(hideAccountEmails.value ? '隐私模式已开启：账号邮箱已隐藏' : '隐私模式已关闭：账号邮箱已显示', 'success')
}

function toggleAccountLayout() {
  accountLayout.value = accountLayout.value === 'dense' ? 'standard' : 'dense'
  oauthPage.value = 1
  apiPage.value = 1
  writeStoredOption('easyllm.openai.accountLayout', accountLayout.value)
}

function setAccountPageSize(value) {
  accountPageSize.value = normalizeAccountPageSize(value)
  oauthPage.value = 1
  apiPage.value = 1
  writeStoredOption('easyllm.openai.accountPageSize', String(accountPageSize.value))
}

function readStoredAccountPageSize(fallback) {
  try {
    return localStorage.getItem('easyllm.openai.accountPageSize') || fallback
  } catch {
    return fallback
  }
}

function normalizeAccountPageSize(value) {
  const size = Math.floor(Number(value))
  if (!Number.isFinite(size) || size <= 0) return PAGE_SIZE
  return Math.min(size, MAX_PAGE_SIZE)
}

function paginationRangeText(page, total, pageSize) {
  if (total <= 0) return '显示 0'
  const start = (Math.max(1, page) - 1) * pageSize + 1
  const end = Math.min(total, start + pageSize - 1)
  return `显示 ${start}-${end} / ${total}`
}

function setAccountSortMode(mode) {
  accountSortMode.value = accountSortModeValues.includes(mode) ? mode : 'smart'
  oauthPage.value = 1
  apiPage.value = 1
  clearOAuthSelection()
  clearAPISelection()
  writeStoredOption('easyllm.openai.accountSortMode', accountSortMode.value)
}

function setPlanGroupFilter(groupID) {
  const allowed = ['all', 'team', 'plus', 'free']
  planGroupFilter.value = allowed.includes(groupID) ? groupID : 'all'
  oauthPage.value = 1
  clearOAuthSelection()
}

function normalizeAccountGroup(group) {
  if (!group || typeof group !== 'object') return null
  const id = String(group.id || '').trim()
  const name = String(group.name || '').trim()
  if (!id || !name) return null
  const ids = Array.isArray(group.account_ids) ? group.account_ids.map(accountId).filter(Boolean) : []
  return {
    id,
    name,
    account_ids: Array.from(new Set(ids))
  }
}

function persistAccountGroups() {
  accountGroups.value = accountGroups.value
    .map(normalizeAccountGroup)
    .filter(Boolean)
  writeStoredJSON('easyllm.openai.accountGroups', accountGroups.value)
  if (activeGroupFilter.value !== 'all' && !accountGroups.value.some(group => group.id === activeGroupFilter.value)) {
    activeGroupFilter.value = 'all'
    persistActiveGroupFilter()
  }
}

function persistActiveGroupFilter() {
  if (activeGroupFilter.value !== 'all' && !accountGroups.value.some(group => group.id === activeGroupFilter.value)) {
    activeGroupFilter.value = 'all'
  }
  oauthPage.value = 1
  writeStoredOption('easyllm.openai.activeGroupFilter', activeGroupFilter.value)
}

function accountGroupNames(account) {
  const id = accountId(account?.id || '')
  if (!id) return []
  return accountGroups.value
    .filter(group => Array.isArray(group.account_ids) && group.account_ids.map(accountId).includes(id))
    .map(group => group.name)
}

function createAccountGroup() {
  const name = newGroupName.value.trim()
  if (!name) {
    showToast('请输入分组名称', 'error')
    return
  }
  const exists = accountGroups.value.some(group => group.name.toLowerCase() === name.toLowerCase())
  if (exists) {
    showToast('分组名称已存在', 'error')
    return
  }
  const id = `group-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
  accountGroups.value = [
    ...accountGroups.value,
    { id, name, account_ids: [] }
  ]
  newGroupName.value = ''
  persistAccountGroups()
  showToast('分组已创建', 'success')
}

function addSelectedToGroup(groupId) {
  if (selectedOAuthIds.value.length === 0) {
    showToast('请先勾选账号', 'error')
    return
  }
  const selected = new Set(selectedOAuthIds.value.map(accountId))
  accountGroups.value = accountGroups.value.map(group => {
    if (group.id !== groupId) return group
    const nextIDs = Array.from(new Set([...(group.account_ids || []).map(accountId), ...selected]))
    return { ...group, account_ids: nextIDs }
  })
  persistAccountGroups()
  showToast(`已加入 ${selected.size} 个账号`, 'success')
}

function removeSelectedFromGroup(groupId) {
  if (selectedOAuthIds.value.length === 0) {
    showToast('请先勾选账号', 'error')
    return
  }
  const selected = new Set(selectedOAuthIds.value.map(accountId))
  accountGroups.value = accountGroups.value.map(group => {
    if (group.id !== groupId) return group
    return {
      ...group,
      account_ids: (group.account_ids || []).map(accountId).filter(id => !selected.has(id))
    }
  })
  persistAccountGroups()
  showToast(`已移出 ${selected.size} 个账号`, 'success')
}

async function deleteAccountGroup(groupId) {
  const group = accountGroups.value.find(item => item.id === groupId)
  if (!group) return
  const confirmed = await requestOperationConfirm({
    title: '删除分组',
    message: `确认删除分组「${group.name}」吗？`,
    details: '账号本身不会被删除，只会移除该分组。',
    confirmText: '删除分组',
    tone: 'danger',
  })
  if (!confirmed) return
  accountGroups.value = accountGroups.value.filter(item => item.id !== groupId)
  persistAccountGroups()
  showToast('分组已删除', 'success')
}

function syncAccountGroupsWithAccounts() {
  const oauthIDSet = new Set(oauthAccounts.value.map(account => accountId(account.id)))
  let changed = false
  accountGroups.value = accountGroups.value
    .map(normalizeAccountGroup)
    .filter(Boolean)
    .map(group => {
      const nextIDs = group.account_ids.filter(id => oauthIDSet.has(id))
      if (nextIDs.length !== group.account_ids.length) changed = true
      return { ...group, account_ids: nextIDs }
    })
  if (changed) persistAccountGroups()
}

function dateSortValue(value) {
  if (!value) return null
  const timestamp = new Date(value).getTime()
  return Number.isFinite(timestamp) ? timestamp : null
}

function compareNullableNumber(left, right, direction = 'desc') {
  if (left == null && right == null) return 0
  if (left == null) return 1
  if (right == null) return -1
  return direction === 'desc' ? right - left : left - right
}

function compareText(left, right, direction = 'asc') {
  const result = String(left || '').localeCompare(String(right || ''), 'zh-Hans-CN', { numeric: true, sensitivity: 'base' })
  return direction === 'desc' ? -result : result
}

function compareActiveAccountFirst(left, right) {
  if (!!left?.is_codex_active === !!right?.is_codex_active) return 0
  return left?.is_codex_active ? -1 : 1
}

function accountCreatedSortValue(account) {
  return dateSortValue(account?.created_at) ?? 0
}

function accountUpdatedSortValue(account) {
  return dateSortValue(account?.updated_at) ?? accountCreatedSortValue(account)
}

function oauthRemainingQuota(account, windowKey) {
  if (windowKey === '5h') {
    if (isTeamPlan(account)) return oauthRemainingQuota(account, '7d')
    return account?.quota_5h_used_percent == null ? null : 100 - Number(account.quota_5h_used_percent)
  }
  if (account?.quota_7d_used_percent != null) return 100 - Number(account.quota_7d_used_percent)
  if (account?.quota_total && account.quota_total > 0) return quotaPct(account)
  return null
}

function accountPlanRank(account) {
  const plan = accountPlanType(account)
  if (plan === 'team') return 3
  if (plan === 'plus') return 2
  if (plan === 'free') return 1
  return 0
}

function sortOAuthAccounts(list) {
  const sorted = [...list]
  sorted.sort((left, right) => {
    const activeDiff = compareActiveAccountFirst(left, right)
    if (activeDiff !== 0) return activeDiff

    let diff = 0
    if (accountSortMode.value === 'updated_desc') {
      diff = compareNullableNumber(accountUpdatedSortValue(left), accountUpdatedSortValue(right), 'desc')
    } else if (accountSortMode.value === 'quota_5h_desc') {
      diff = compareNullableNumber(oauthRemainingQuota(left, '5h'), oauthRemainingQuota(right, '5h'), 'desc')
    } else if (accountSortMode.value === 'quota_5h_asc') {
      diff = compareNullableNumber(oauthRemainingQuota(left, '5h'), oauthRemainingQuota(right, '5h'), 'asc')
    } else if (accountSortMode.value === 'plan_desc') {
      diff = compareNullableNumber(accountPlanRank(left), accountPlanRank(right), 'desc')
    } else if (accountSortMode.value === 'plan_asc') {
      diff = compareNullableNumber(accountPlanRank(left), accountPlanRank(right), 'asc')
    } else if (accountSortMode.value === 'email_asc') {
      diff = compareText(left?.email || left?.id, right?.email || right?.id, 'asc')
    } else {
      diff = oauthAccountPriority(left) - oauthAccountPriority(right)
      if (diff === 0) diff = compareNullableNumber(oauthRemainingQuota(left, '5h'), oauthRemainingQuota(right, '5h'), 'desc')
      if (diff === 0) diff = compareNullableNumber(accountUpdatedSortValue(left), accountUpdatedSortValue(right), 'desc')
    }
    if (diff === 0) diff = compareNullableNumber(accountUpdatedSortValue(left), accountUpdatedSortValue(right), 'desc')
    if (diff === 0) diff = compareNullableNumber(accountCreatedSortValue(left), accountCreatedSortValue(right), 'desc')
    if (diff !== 0) return diff
    return compareText(left?.email || left?.id, right?.email || right?.id, 'asc')
  })
  return sorted
}

function sortAPIAccounts(list) {
  const sorted = [...list]
  sorted.sort((left, right) => {
    const activeDiff = compareActiveAccountFirst(left, right)
    if (activeDiff !== 0) return activeDiff

    let diff = compareNullableNumber(accountUpdatedSortValue(left), accountUpdatedSortValue(right), 'desc')
    if (diff === 0) diff = compareNullableNumber(accountCreatedSortValue(left), accountCreatedSortValue(right), 'desc')
    if (diff !== 0) return diff
    return compareText(left?.model_provider || left?.email || left?.id, right?.model_provider || right?.email || right?.id, 'asc')
  })
  return sorted
}

// Methods
async function loadAccounts() {
  loading.value = true
  try {
    // api interceptor returns response.data directly, so res IS the array
    const res = await api.get('/openai/accounts')
    accounts.value = Array.isArray(res) ? res : (res || [])
    syncAccountGroupsWithAccounts()
    pruneSelectedAccountIds()
  } catch (e) {
    showToast('加载账号失败: ' + e.message, 'error')
  } finally {
    loading.value = false
  }
}

async function switchAccount(account) {
  switchingId.value = account.id
  try {
    await api.post(`/openai/accounts/${account.id}/switch`)
    accounts.value.forEach(a => { a.is_codex_active = (a.id === account.id) })
    const idx = accounts.value.findIndex(a => a.id === account.id)
    if (idx > 0) {
      const [item] = accounts.value.splice(idx, 1)
      accounts.value.unshift(item)
    }
    showToast(`已切换到 ${accountDisplayLabel(account)}，~/.codex/auth.json 已更新`, 'success')
  } catch (e) {
    showToast('切换失败: ' + (e.response?.data?.error || e.message), 'error')
  } finally {
    switchingId.value = null
  }
}

async function switchAPIAccount(account) {
  switchingId.value = account.id
  try {
    await api.post(`/openai/api-accounts/${account.id}/switch`)
    accounts.value.forEach(a => { a.is_codex_active = (a.id === account.id) })
    const idx = accounts.value.findIndex(a => a.id === account.id)
    if (idx > 0) {
      const [item] = accounts.value.splice(idx, 1)
      accounts.value.unshift(item)
    }
    showToast(`已切换到 ${accountDisplayLabel(account)}，~/.codex/config.toml 已更新`, 'success')
  } catch (e) {
    showToast('切换失败: ' + (e.response?.data?.error || e.message), 'error')
  } finally {
    switchingId.value = null
  }
}

async function testAPIAccount(account) {
  testingAPIId.value = account.id
  try {
    const res = await api.post(`/openai/api-accounts/${account.id}/test`)
    if (res.success) {
      showToast(`测活成功 · ${res.http_status} · ${res.latency_ms}ms`, 'success')
    } else {
      showToast(`测活失败: ${res.error || 'HTTP ' + res.http_status}`, 'error')
    }
  } catch (e) {
    showToast('测活失败: ' + (e.response?.data?.error || e.message), 'error')
  } finally {
    testingAPIId.value = null
  }
}

async function refreshAllTokens() {
  if (oauthAccounts.value.length === 0) {
    showToast('没有可刷新的 OAuth 账号', 'error')
    return
  }
  const confirmed = await requestOperationConfirm({
    title: '刷新全部 Token',
    message: '刷新全部 Token 会轮换 refresh_token。',
    details: '确认后会等待全部账号刷新完成并写入本地库，然后自动下载最新账号 JSON。',
    confirmText: '继续刷新',
    tone: 'warning',
  })
  if (!confirmed) return
  refreshingAllTokens.value = true
  try {
    const res = await openaiAPI.refreshAll()
    const success = res?.success ?? 0
    const skipped = res?.skipped ?? 0
    const failed = res?.failed ?? 0
    const parts = []
    if (success > 0) parts.push(`成功 ${success}`)
    if (skipped > 0) parts.push(`跳过 ${skipped}`)
    if (failed > 0) parts.push(`失败 ${failed}`)
    await Promise.all([loadAccounts(), loadServiceConfig(), loadLocalAccess()])
    if (success > 0) {
      await exportAccounts({ showSuccessToast: false, throwOnError: true })
      showToast(`全部刷新完成并已导出最新 JSON：${parts.join('，')}`, failed > 0 ? 'error' : 'success')
    } else {
      showToast(`全部刷新完成：${parts.join('，') || '无可用账号'}，没有成功刷新账号，未自动下载 JSON`, 'error')
    }
  } catch (e) {
    showToast('刷新全部失败: ' + e.message, 'error')
  } finally {
    refreshingAllTokens.value = false
  }
}

async function refreshToken(account) {
  refreshingId.value = account.id
  try {
    await api.post(`/openai/accounts/${account.id}/refresh-token`)
    await loadAccounts()
    await exportAccounts({ showSuccessToast: false, throwOnError: true })
    showToast(`${accountDisplayLabel(account)} token 刷新成功，已导出最新 JSON`, 'success')
  } catch (e) {
    showToast('刷新失败: ' + e.message, 'error')
  } finally {
    refreshingId.value = null
  }
}

async function deleteAccount(id) {
  const target = accounts.value.find(a => a.id === id)
  if (!target) {
    showToast('找不到该账号', 'error')
    return
  }
  deleteTargetId.value = id
  deleteTargetLabel.value = accountDisplayLabel(target)
  showDeleteConfirm.value = true
}

function closeDeleteConfirm() {
  if (deletingAccount.value) return
  resetDeleteConfirm()
}

function resetDeleteConfirm() {
  deletingAccount.value = false
  showDeleteConfirm.value = false
  deleteTargetId.value = null
  deleteTargetLabel.value = ''
}

async function confirmDeleteAccount() {
  const id = deleteTargetId.value
  if (!id) return
  deletingAccount.value = true
  try {
    await api.delete(`/openai/accounts/${id}`)
    // Remove locally for instant push
    accounts.value = accounts.value.filter(a => String(a.id) !== String(id))
    if (activeTab.value === 'oauth' && oauthPage.value > oauthTotalPages.value) {
      oauthPage.value = oauthTotalPages.value
    }
    if (activeTab.value === 'api' && apiPage.value > apiTotalPages.value) {
      apiPage.value = apiTotalPages.value
    }
    showToast('已删除', 'success')
    resetDeleteConfirm()
    // Refresh list from server as fail-safe
    await loadAccounts()
  } catch (e) {
    showToast('删除失败: ' + (e.response?.data?.error || e.message), 'error')
    deletingAccount.value = false
  }
}

async function toggleProxy(account) {
  togglingProxyId.value = account.id
  try {
    const res = await api.post(`/openai/accounts/${account.id}/toggle-proxy`)
    const idx = accounts.value.findIndex(a => a.id === account.id)
    if (idx >= 0) accounts.value[idx].proxy_enabled = res.proxy_enabled
    const label = accountDisplayLabel(account)
    showToast(res.proxy_enabled ? `${label} 已加入代理池` : `${label} 已移出代理池`, 'success')
  } catch (e) {
    showToast('操作失败: ' + e.message, 'error')
  } finally {
    togglingProxyId.value = null
  }
}

async function toggleProxyAll(enabled) {
  if (oauthAccounts.value.length === 0) return
  togglingProxyAll.value = true
  try {
    const res = await api.post('/openai/accounts/toggle-proxy-all', { enabled })
    const count = res?.updated_count ?? 0
    accounts.value.forEach(a => { if (!a.account_type || a.account_type === 'oauth') a.proxy_enabled = enabled })
    showToast(enabled ? `${count} 个账号已加入代理池，/v1/* 轮询已开启` : `${count} 个账号已移出代理池`, 'success')
  } catch (e) {
    showToast('一键操作失败: ' + (e.response?.data?.error || e.message), 'error')
  } finally {
    togglingProxyAll.value = false
  }
}

// ---- Import examples ----

const exampleFiles = {
  'token-files': {
    filename: 'token_example.json',
    content: JSON.stringify({
      "id_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0...",
      "access_token": "eyJhbGciOiJSUzI1NiIsImtpZCI6IjE5MzQ0ZTY1In0.eyJhdWQiOlsiaHR0cHM6Ly9hcGkub3BlbmFpLmNvbSJdfQ...",
      "refresh_token": "v1.MjQ3NDUzMTg3NjE0NzY3OTc0NjQxNDExNDY3ODk...",
      "email": "your-email@example.com",
      "chatgpt_account_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
      "expires_at": 1772632299
    }, null, 2)
  },
  'auto-files': {
    filename: 'token_account1.json',
    content: JSON.stringify({
      "id_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
      "access_token": "eyJhbGciOiJSUzI1NiIsImtpZCI6IjE5MzQ0ZTY1In0...",
      "refresh_token": "v1.MjQ3NDUzMTg3NjE0NzY3OTc0NjQxNDExNDY3ODk...",
      "email": "account1@example.com",
      "chatgpt_account_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
      "expires_at": 1772632299
    }, null, 2)
  },
  'refresh-tokens': {
    filename: 'refresh_tokens_example.json',
    content: JSON.stringify([
      "v1.MjQ3NDUzMTg3NjE0NzY3OTc0NjQxNDExNDY3ODk...",
      "v1.OTg3NjU0MzIxMDk4NzY1NDMyMTA5ODc2NTQzMjE...",
      "v1.NTY3ODkwMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM..."
    ], null, 2)
  },
  'cpa': {
    filename: 'example-cpa.json',
    content: JSON.stringify({
      type: 'codex',
      email: 'your-email@example.com',
      expired: '2026-06-01T00:00:00Z',
      account_id: 'xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx',
      access_token: 'eyJhbGciOiJSUzI1NiIsImtpZCI6IjE5MzQ0ZTY1In0...',
      refresh_token: 'rt_example_refresh_token',
      id_token: 'eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...',
      last_refresh: '2026-05-08T12:00:00Z',
      plan_type: 'plus'
    }, null, 2)
  },
  'from-export': {
    filename: 'easyllm_accounts_backup_example.json',
    content: JSON.stringify({
      "exported_at": "2026-05-08T12:00:00Z",
      "version": "2.0.0",
      "oauth_accounts": [
        {
          "email": "your-email@example.com",
          "account_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
          "access_token": "eyJhbGciOiJSUzI1NiIsImtpZCI6IjE5MzQ0ZTY1In0...",
          "refresh_token": "v1.MjQ3NDUzMTg3NjE0NzY3OTc0NjQxNDExNDY3ODk...",
          "id_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
          "expired": "2026-06-01T00:00:00Z",
          "last_refresh": "2026-05-08T12:00:00Z",
          "type": "codex"
        }
      ],
      "api_accounts": [],
      "local_access": {
        "enabled": true,
        "port": 8022,
        "routing_strategy": "auto",
        "restrict_free_accounts": true,
        "account_ids": []
      }
    }, null, 2)
  }
}

function downloadExample(mode) {
  const example = exampleFiles[mode]
  if (!example) return
  const blob = new Blob([example.content], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = example.filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

// ---- Import ----

function selectImportMode(mode) {
  importMode.value = mode
  importResults.value = null
  importFiles.value = []
  importTokens.value = []
  importCPAFiles.value = []
  importCPAAccountCount.value = 0
  importAutoFiles.value = []
  importBackupFile.value = null
}

const canRunImport = computed(() => {
  if (importMode.value === 'token-files') return importFiles.value.length > 0
  if (importMode.value === 'auto-files') return importAutoFiles.value.length > 0
  if (importMode.value === 'refresh-tokens') return importTokens.value.length > 0
  if (importMode.value === 'cpa') return importCPAFiles.value.length > 0
  if (importMode.value === 'from-export') return !!importBackupFile.value
  return false
})

const importButtonLabel = computed(() => {
  if (importMode.value === 'token-files') return `导入 ${importFiles.value.length} 个文件`
  if (importMode.value === 'auto-files') return `自适应导入 ${importAutoFiles.value.length} 个文件`
  if (importMode.value === 'refresh-tokens') return `导入 ${importTokens.value.length} 个账号`
  if (importMode.value === 'cpa') return `导入 ${importCPAAccountCount.value} 个 CPA 账号`
  if (importMode.value === 'from-export') {
    const total = (importBackupFile.value?.oauth_accounts?.length ?? 0) + (importBackupFile.value?.api_accounts?.length ?? 0)
    return importBackupFile.value?.local_access ? `从备份导入 ${total} 个账号 + 本地服务配置` : `从备份导入 ${total} 个账号`
  }
  return '导入'
})

function handleFiles(files) {
  if (!files.length) return
  importFiles.value = Array.from(files)
  importResults.value = null
}

function handleAutoFiles(files) {
  if (!files.length) return
  const jsonFiles = Array.from(files).filter(f => /\.json$/i.test(f.name))
  if (!jsonFiles.length) {
    showToast('请选择 .json 文件', 'error')
    return
  }
  importAutoFiles.value = jsonFiles
  importResults.value = null
}

function handleAutoFileSelect(event) {
  handleAutoFiles(event.target.files)
  event.target.value = ''
}

function handleAutoDirectorySelect(event) {
  handleAutoFiles(event.target.files)
  event.target.value = ''
}

function openImportDirectoryPicker() {
  const input = importAutoDirectoryInput.value
  if (!input) {
    showToast('当前环境不支持选择文件夹，请改选 JSON 文件', 'error')
    return
  }

  selectingImportDirectory.value = true
  try {
    input.click()
  } finally {
    window.setTimeout(() => {
      selectingImportDirectory.value = false
    }, 300)
  }
}

function resetScanImportSelection() {
  importAutoFiles.value = []
  importResults.value = null
}

function resolveCPAAccountCount(data) {
  if (Array.isArray(data)) return data.length
  if (data && typeof data === 'object') {
    if (data.access_token || data.refresh_token || data.id_token) return 1
  }
  return 0
}

async function countCPAAccountsInFile(file) {
  const text = await file.text()
  const trimmed = text.trim()
  if (!trimmed) return 0
  if (trimmed.startsWith('[')) {
    const data = JSON.parse(trimmed)
    return resolveCPAAccountCount(data)
  }
  let count = 0
  const lines = trimmed.split(/\r?\n/).map(line => line.trim()).filter(Boolean)
  if (lines.length > 1) {
    for (const line of lines) {
      count += resolveCPAAccountCount(JSON.parse(line))
    }
    return count
  }
  return resolveCPAAccountCount(JSON.parse(trimmed))
}

async function handleCPAFiles(files) {
  if (!files.length) return
  const items = []
  let total = 0
  for (const file of Array.from(files)) {
    try {
      const count = await countCPAAccountsInFile(file)
      if (count <= 0) {
        showToast(`文件 ${file.name} 中未找到有效 CPA 账号`, 'error')
        continue
      }
      items.push({ name: file.name, file, count })
      total += count
    } catch (err) {
      showToast(`解析 ${file.name} 失败: ${err.message}`, 'error')
    }
  }
  if (!items.length) return
  importCPAFiles.value = items
  importCPAAccountCount.value = total
  importResults.value = null
}

function handleCPAFileSelect(event) {
  handleCPAFiles(event.target.files)
  event.target.value = ''
}

function handleMultiFileSelect(event) {
  handleFiles(event.target.files)
  event.target.value = ''
}

function parseBackupFile(file) {
  if (!file) return
  const reader = new FileReader()
  reader.onload = (e) => {
    try {
      const data = JSON.parse(e.target.result)
      if (!data.oauth_accounts && !data.api_accounts && !data.local_access) {
        showToast('文件格式不正确，请上传由"导出账号"生成的备份文件', 'error')
        return
      }
      importBackupFile.value = data
      importResults.value = null
    } catch (err) {
      showToast('文件解析失败: ' + err.message, 'error')
    }
  }
  reader.readAsText(file)
}

function parseRefreshTokenFile(file) {
  if (!file) return
  const reader = new FileReader()
  reader.onload = (e) => {
    try {
      const raw = String(e.target.result ?? '')
      let tokens = []
      try {
        const data = JSON.parse(raw)
        tokens = Array.isArray(data) ? data : [data]
      } catch {
        tokens = raw.split(/\r?\n|,/)
      }
      const valid = tokens
        .filter(t => typeof t === 'string' && t.trim().length > 0)
        .map(t => t.trim())
      if (valid.length === 0) {
        showToast('文件中没有有效的 refresh_token', 'error')
        return
      }
      importTokens.value = valid
      importResults.value = null
    } catch (err) {
      showToast('文件解析失败: ' + err.message, 'error')
    }
  }
  reader.readAsText(file)
}

function handleDrop(event) {
  dragging.value = false
  const files = Array.from(event.dataTransfer?.files || [])
  if (!files.length) return

  if (importMode.value === 'token-files') {
    handleFiles(files)
    return
  }
  if (importMode.value === 'cpa') {
    handleCPAFiles(files)
    return
  }
  if (importMode.value === 'auto-files') {
    handleAutoFiles(files)
    return
  }

  const [file] = files
  if (importMode.value === 'refresh-tokens') {
    parseRefreshTokenFile(file)
    return
  }
  if (importMode.value === 'from-export') {
    parseBackupFile(file)
  }
}

function handleBackupFileSelect(event) {
  const file = event.target.files?.[0]
  parseBackupFile(file)
  event.target.value = ''
}

function handleFileSelect(event) {
  const file = event.target.files?.[0]
  parseRefreshTokenFile(file)
  event.target.value = ''
}

async function runImport() {
  importing.value = true
  importResults.value = null
  try {
    let res

    if (importMode.value === 'token-files') {
      // Upload multiple JSON files via multipart form
      // Use fetch directly to avoid Axios default Content-Type overriding multipart boundary
      const formData = new FormData()
      for (const f of importFiles.value) {
        formData.append('files', f)
      }
      const token = localStorage.getItem('easyllm_token')
      const fetchRes = await fetch('/api/v1/openai/import/token-files', {
        method: 'POST',
        body: formData,
        headers: token ? { 'Authorization': `Bearer ${token}` } : {}
      })
      if (!fetchRes.ok) {
        const errData = await fetchRes.json().catch(() => ({}))
        throw new Error(errData.error || `HTTP ${fetchRes.status}`)
      }
      res = await fetchRes.json()
      // Note: api interceptor returns response.data directly, so res IS the data object
      importResults.value = {
        success: res?.success ?? 0,
        skipped: res?.skipped ?? 0,
        failed:  res?.failed  ?? 0,
        total:   res?.total   ?? 0,
        results: res?.results ?? []
      }

    } else if (importMode.value === 'auto-files') {
      const formData = new FormData()
      for (const f of importAutoFiles.value) {
        formData.append('files', f)
      }
      const token = localStorage.getItem('easyllm_token')
      const fetchRes = await fetch('/api/v1/openai/import/auto-files', {
        method: 'POST',
        body: formData,
        headers: token ? { Authorization: `Bearer ${token}` } : {}
      })
      if (!fetchRes.ok) {
        const errData = await fetchRes.json().catch(() => ({}))
        throw new Error(errData.error || `HTTP ${fetchRes.status}`)
      }
      res = await fetchRes.json()
      importResults.value = {
        success: res?.success ?? 0,
        skipped: res?.skipped ?? 0,
        failed: res?.failed ?? 0,
        total: res?.total ?? 0,
        results: res?.results ?? []
      }

    } else if (importMode.value === 'from-export') {
      // 直接消费备份 JSON，无需 API 调用
      res = await api.post('/openai/import/from-export', {
        oauth_accounts: importBackupFile.value.oauth_accounts || [],
        api_accounts:   importBackupFile.value.api_accounts   || [],
        local_access:   importBackupFile.value.local_access   || null,
      })
      importResults.value = {
        success: res?.success ?? 0,
        skipped: res?.skipped ?? 0,
        failed:  res?.failed  ?? 0,
        total:   res?.total   ?? 0,
        results: res?.results ?? []
      }

    } else if (importMode.value === 'cpa') {
      const formData = new FormData()
      for (const item of importCPAFiles.value) {
        formData.append('files', item.file)
      }
      const token = localStorage.getItem('easyllm_token')
      const fetchRes = await fetch('/api/v1/openai/import/cpa', {
        method: 'POST',
        body: formData,
        headers: token ? { Authorization: `Bearer ${token}` } : {}
      })
      if (!fetchRes.ok) {
        const errData = await fetchRes.json().catch(() => ({}))
        throw new Error(errData.error || `HTTP ${fetchRes.status}`)
      }
      res = await fetchRes.json()
      importResults.value = {
        success: res?.success ?? 0,
        skipped: res?.skipped ?? 0,
        failed: res?.failed ?? 0,
        total: res?.total ?? 0,
        results: res?.results ?? []
      }

    } else {
      // Legacy: refresh_token list requires OpenAI API calls
      res = await api.post('/openai/import/refresh-tokens', {
        refresh_tokens: importTokens.value
      })
      importResults.value = {
        success: res?.successful ?? 0,
        skipped: 0,
        failed:  res?.failed ?? 0,
        total:   res?.total  ?? 0,
        results: (res?.results ?? []).map(r => ({
          ...r,
          filename: r.token_preview,
          skipped: false
        }))
      }
    }

    const restoredLocalAccess = importMode.value === 'from-export' && !!importBackupFile.value?.local_access
    const importedCount = importResults.value?.success ?? 0
    if (restoredLocalAccess || importedCount > 0) {
      const reloadTasks = [loadServiceConfig(), loadLocalAccess()]
      if (importedCount > 0) reloadTasks.push(loadAccounts())
      await Promise.all(reloadTasks)
    }

    if (importedCount > 0) {
      showToast(restoredLocalAccess ? `成功导入 ${importResults.value.success} 个账号，并恢复本地服务配置` : `成功导入 ${importResults.value.success} 个账号`, 'success')
    } else if (restoredLocalAccess) {
      showToast('本地服务配置已恢复', 'success')
    } else if (importResults.value?.total > 0 && importResults.value?.failed === 0) {
      showToast('所有账号已存在，跳过重复导入', 'error')
    }
  } catch (e) {
    showToast('导入失败: ' + (e.message || String(e)), 'error')
  } finally {
    importing.value = false
  }
}

function closeImportDialog() {
  if (importing.value) return
  showImportDialog.value = false
  importTokens.value = []
  importFiles.value = []
  importBackupFile.value = null
  importCPAFiles.value = []
  importCPAAccountCount.value = 0
  importAutoFiles.value = []
  importResults.value = null
}

// ---- OAuth ----
function createEmptyOAuthState() {
  return {
    authUrl: '',
    sessionId: '',
    manualInput: '',
    loading: false,
    error: '',
    autoCallbackEnabled: false,
    autoCallbackError: '',
    status: 'idle'
  }
}

function resetOAuthState() {
  stopOAuthPolling()
  oauthState.value = createEmptyOAuthState()
}

function stopOAuthPolling() {
  if (oauthPollTimer) {
    clearTimeout(oauthPollTimer)
    oauthPollTimer = null
  }
}

async function cancelOAuthSession(sessionId) {
  if (!sessionId) return
  try {
    await openaiAPI.cancelOAuthSession(sessionId)
  } catch {
    // best effort cleanup
  }
}

function openOAuthDialog() {
  showOAuthDialog.value = true
  resetOAuthState()
  generateOAuthUrl()
}

async function closeOAuthDialog() {
  const sessionId = oauthState.value.sessionId
  showOAuthDialog.value = false
  resetOAuthState()
  await cancelOAuthSession(sessionId)
}

async function generateOAuthUrl() {
  const previousSessionId = oauthState.value.sessionId
  stopOAuthPolling()
  oauthState.value.loading = true
  oauthState.value.error = ''
  oauthState.value.autoCallbackError = ''

  const popup = window.open('', '_blank', 'noopener,noreferrer')
  try {
    if (previousSessionId) {
      await cancelOAuthSession(previousSessionId)
    }
    const res = await openaiAPI.generateOAuthUrl()
    oauthState.value.authUrl = res.auth_url
    oauthState.value.sessionId = res.session_id
    oauthState.value.manualInput = ''
    oauthState.value.autoCallbackEnabled = !!res.auto_callback_enabled
    oauthState.value.autoCallbackError = res.auto_callback_error || ''
    oauthState.value.status = res.auto_callback_enabled ? 'pending' : 'manual'

    if (popup) {
      popup.location = res.auth_url
    } else {
      const opened = window.open(res.auth_url, '_blank', 'noopener,noreferrer')
      if (!opened && !oauthState.value.autoCallbackError) {
        oauthState.value.autoCallbackError = '浏览器未能自动打开，请点击“打开”按钮或复制链接。'
      }
    }

    if (res.auto_callback_enabled) {
      scheduleOAuthPoll()
    }
  } catch (e) {
    popup?.close?.()
    oauthState.value.error = '生成失败: ' + e.message
  } finally {
    oauthState.value.loading = false
  }
}

function copyAuthUrl() {
  navigator.clipboard.writeText(oauthState.value.authUrl)
  showToast('链接已复制', 'success')
}

function openOAuthInBrowser() {
  if (!oauthState.value.authUrl) return
  const opened = window.open(oauthState.value.authUrl, '_blank', 'noopener,noreferrer')
  if (!opened) {
    showToast('浏览器未能自动打开，已复制授权链接', 'success')
    copyAuthUrl()
  }
}

function scheduleOAuthPoll() {
  stopOAuthPolling()
  oauthPollTimer = window.setTimeout(pollOAuthSession, OAUTH_POLL_INTERVAL_MS)
}

async function pollOAuthSession() {
  if (!showOAuthDialog.value || !oauthState.value.sessionId || !oauthState.value.autoCallbackEnabled) return

  try {
    const res = await openaiAPI.getOAuthSession(oauthState.value.sessionId)
    oauthState.value.status = res.status || 'pending'

    if (res.status === 'error') {
      stopOAuthPolling()
      oauthState.value.error = res.error || 'OAuth 授权失败'
      return
    }

    if (res.status === 'callback_received') {
      stopOAuthPolling()
      await exchangeOAuthCode()
      return
    }
  } catch (e) {
    stopOAuthPolling()
    oauthState.value.error = /expired/i.test(e.message)
      ? '授权会话已过期，请重新发起登录'
      : `授权状态检查失败: ${e.message}`
    return
  }

  scheduleOAuthPoll()
}

async function exchangeOAuthCode() {
  stopOAuthPolling()
  oauthState.value.loading = true
  oauthState.value.error = ''
  try {
    const manualInput = oauthState.value.manualInput.trim()
    const payload = { session_id: oauthState.value.sessionId }
    if (manualInput) {
      if (manualInput.includes('://') || manualInput.startsWith('/') || manualInput.startsWith('?') || manualInput.includes('code=')) {
        payload.callback_url = manualInput
      } else {
        payload.code = manualInput
      }
	}
	const res = await openaiAPI.exchangeOAuthCode(payload)
	await Promise.all([loadAccounts(), loadServiceConfig(), loadLocalAccess()])
	showOAuthDialog.value = false
    resetOAuthState()
    const email = res?.account?.email || ''
    if (res?.auto_joined_proxy) {
      showToast(email && !hideAccountEmails.value ? `${email} 已登录并自动加入代理池` : 'OAuth 登录成功，账号已自动加入代理池', 'success')
    } else {
      showToast(email && !hideAccountEmails.value ? `${email} 已登录` : 'OAuth 登录成功', 'success')
    }
  } catch (e) {
    oauthState.value.error = e.message
  } finally {
    oauthState.value.loading = false
  }
}

// ---- API Account ----
function providerDisplayName(provider) {
  const key = String(provider || '').trim().toLowerCase()
  return providerDisplayNames[key] || provider || 'API'
}

function createAPIForm() {
  return { model_provider: '', model: '', base_url: '', api_key: '', wire_api: 'responses', model_reasoning_effort: '' }
}

function openAddAPIDialog() {
  editingAPIAccount.value = null
  apiForm.value = createAPIForm()
  apiFormError.value = ''
  showAddAPIDialog.value = true
}

function editAPIAccount(account) {
  editingAPIAccount.value = account
  apiForm.value = {
    model_provider: account.model_provider || '',
    model: account.model || '',
    base_url: account.base_url || '',
    api_key: '',
    wire_api: account.wire_api || 'responses',
    model_reasoning_effort: account.model_reasoning_effort || ''
  }
  apiFormError.value = ''
  showAddAPIDialog.value = true
}

function closeAPIDialog() {
  showAddAPIDialog.value = false
  editingAPIAccount.value = null
  apiForm.value = createAPIForm()
  apiFormError.value = ''
}

async function saveAPIAccount() {
  if (!apiForm.value.model_provider || !apiForm.value.model || !apiForm.value.base_url) {
    apiFormError.value = 'model_provider、model 和 base_url 为必填项'
    return
  }
  savingAPI.value = true
  apiFormError.value = ''
  try {
    const payload = { ...apiForm.value }
    if (!payload.model_reasoning_effort) payload.model_reasoning_effort = null
    if (editingAPIAccount.value) {
      const res = await api.put(`/openai/api-accounts/${editingAPIAccount.value.id}`, payload)
      const idx = accounts.value.findIndex(a => a.id === editingAPIAccount.value.id)
      if (idx >= 0) accounts.value[idx] = res
    } else {
      const res = await api.post('/openai/api-accounts', payload)
      accounts.value.unshift(res)
    }
    closeAPIDialog()
    showToast('保存成功', 'success')
  } catch (e) {
    apiFormError.value = e.message
  } finally {
    savingAPI.value = false
  }
}

// ---- Service Config ----
async function loadServiceConfig() {
  try {
    const res = await api.get('/openai/service-config')
    Object.assign(serviceConfig.value, res)
    serviceApiKeyInput.value = ''
  } catch (e) {
    console.error('Failed to load service config:', e)
  }
}

function applyLocalAccessState(state) {
  if (!state) return
  localAccess.value = {
    ...localAccess.value,
    ...state,
    collection: {
      ...localAccess.value.collection,
      ...(state.collection || {})
    },
    stats: {
      daily: state.stats?.daily || { totals: {}, accounts: [] },
      weekly: state.stats?.weekly || { totals: {}, accounts: [] },
      monthly: state.stats?.monthly || { totals: {}, accounts: [] }
    }
  }
  localAccessSelectedIds.value = [...(localAccess.value.collection?.account_ids || [])].map(accountId)
  localAccessPortInput.value = String(localAccess.value.collection?.port || 8022)
  localAccessRestrictFree.value = localAccess.value.collection?.restrict_free_accounts !== false
}

async function loadLocalAccess() {
  try {
    const res = await api.get('/openai/local-access')
    applyLocalAccessState(res)
  } catch (e) {
    console.error('Failed to load local access:', e)
  }
}



async function openServiceConfig() {
  showServiceConfigDialog.value = true
  await Promise.all([loadServiceConfig(), loadLocalAccess()])
}

async function exportAccounts(options = {}) {
  const showSuccessToast = options?.showSuccessToast !== false
  const throwOnError = options?.throwOnError === true
  exportingAccounts.value = true
  try {
    const payload = await openaiAPI.exportJSON()
    const blob = new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `easyllm-accounts-${new Date().toISOString().slice(0, 10)}.json`
    a.click()
    URL.revokeObjectURL(url)
    if (showSuccessToast) {
      showToast(`已导出 ${payload.oauth_accounts?.length ?? 0} 个 OAuth + ${payload.api_accounts?.length ?? 0} 个 API 账号（基于后端最新落库数据）`, 'success')
    }
    return payload
  } catch (e) {
    showToast('导出失败: ' + e.message, 'error')
    if (throwOnError) throw e
    return null
  } finally {
    exportingAccounts.value = false
  }
}

async function toggleServiceProxyPool() {
  savingServiceConfig.value = true
  try {
    const res = await api.put('/openai/service-config', { proxy_pool_enabled: !serviceConfig.value.proxy_pool_enabled })
    Object.assign(serviceConfig.value, res)
    showToast(serviceConfig.value.proxy_pool_enabled ? '代理池已开启' : '代理池已关闭', 'success')
  } catch (e) {
    showToast('操作失败: ' + e.message, 'error')
  } finally {
    savingServiceConfig.value = false
  }
}

async function updateServiceStrategy(strategy) {
  savingServiceConfig.value = true
  try {
    const res = await api.put('/openai/service-config', { strategy })
    Object.assign(serviceConfig.value, res)
    showToast('轮询策略已更新', 'success')
  } catch (e) {
    showToast('操作失败: ' + e.message, 'error')
  } finally {
    savingServiceConfig.value = false
  }
}

async function activateCodexAPIService() {
  savingServiceConfig.value = true
  try {
    const res = await api.post('/openai/service-config/activate-codex')
    Object.assign(serviceConfig.value, res)
    await Promise.all([loadLocalAccess(), loadAccounts()])
    showToast(res?.codex_app_restarted ? 'Codex 已重启并注入配置' : 'Codex 已启动并注入配置', 'success')
  } catch (e) {
    showToast('启动 Codex API 服务失败: ' + e.message, 'error')
  } finally {
    savingServiceConfig.value = false
  }
}

function toggleLocalAccessAccount(id) {
  const key = accountId(id)
  const account = oauthAccounts.value.find(item => accountId(item.id) === key)
  if (!isLocalAccessEligibleAccount(account)) {
    return
  }
  if (localAccessSelectedIds.value.includes(key)) {
    localAccessSelectedIds.value = localAccessSelectedIds.value.filter(item => item !== key)
  } else {
    localAccessSelectedIds.value = [...localAccessSelectedIds.value, key]
  }
}

function isLocalAccessEligibleAccount(account) {
  return Number(account?._quota_http_status) === 200 && !account?.quota_is_forbidden
}

function quotaStatusBadgeClass(status) {
  const code = Number(status)
  if (code === 401) return 'quota-status-badge--401'
  if (code === 403) return 'quota-status-badge--403'
  if (code === 429) return 'quota-status-badge--429'
  if (code === 503) return 'quota-status-badge--503'
  if (code >= 500) return 'quota-status-badge--5xx'
  if (code >= 400) return 'quota-status-badge--4xx'
  return 'quota-status-badge--other'
}

function selectAllLocalAccessAccounts() {
  localAccessSelectedIds.value = localAccessEligibleAccounts.value.map(account => accountId(account.id)).filter(Boolean)
}

function clearLocalAccessAccounts() {
  localAccessSelectedIds.value = []
}

async function saveAllLocalAccessAccounts() {
  selectAllLocalAccessAccounts()
  if (localAccessSelectedIds.value.length === 0) {
    showToast('暂无 200 成功的 OAuth 账号，请先查询配额', 'error')
    return
  }
  await saveLocalAccessAccounts()
}

async function localAccessAction(task, successText) {
  localAccessBusy.value = true
  try {
    const res = await task()
    applyLocalAccessState(res?.state || res)
    await loadServiceConfig()
    showToast(successText, 'success')
  } catch (e) {
    showToast(e.message || '操作失败', 'error')
    throw e
  } finally {
    localAccessBusy.value = false
  }
}

async function activateLocalAccess() {
  await localAccessAction(
    () => api.post('/openai/local-access/activate'),
    'Codex 本地 API 服务已启动并注入配置'
  )
}

async function deactivateLocalAccess() {
  await localAccessAction(
    () => api.post('/openai/local-access/deactivate'),
    'Codex 本地 API 服务已停止，已移除 EasyLLM 注入配置'
  )
}

async function saveLocalAccessAccounts() {
  await localAccessAction(
    () => api.put('/openai/local-access/accounts', {
      account_ids: localAccessSelectedIds.value,
      restrict_free_accounts: localAccessRestrictFree.value
    }),
    'Codex API 服务账号集合已保存'
  )
}

async function updateLocalAccessRouting(strategy) {
  await localAccessAction(
    () => api.put('/openai/local-access/routing', { strategy }),
    'Codex API 服务调度策略已更新'
  )
}

async function rotateLocalAccessKey() {
  const confirmed = await requestOperationConfirm({
    title: '重置服务 Key',
    message: '重置后当前 Codex API 服务 Key 会立即失效。',
    details: '需要更新外部调用方使用的新 Key 后才能继续访问服务。',
    confirmText: '重置 Key',
    tone: 'danger',
  })
  if (!confirmed) return
  await localAccessAction(
    () => api.post('/openai/local-access/rotate-key'),
    'Codex API 服务 Key 已重置并重新注入'
  )
}

async function clearLocalAccessStats() {
  const confirmed = await requestOperationConfirm({
    title: '清空统计数据',
    message: '确认清空旧版 Codex API 服务统计数据吗？',
    details: '此操作只影响旧版本地统计记录，不会删除账号或服务配置。',
    confirmText: '清空统计',
    tone: 'danger',
  })
  if (!confirmed) return
  await localAccessAction(
    () => api.delete('/openai/local-access/stats'),
    'Codex API 服务统计已清空'
  )
}

async function saveServiceApiKey() {
  savingServiceConfig.value = true
  try {
    const res = await api.put('/openai/service-config', { api_key: serviceApiKeyInput.value })
    Object.assign(serviceConfig.value, res)
    serviceApiKeyInput.value = ''
    showToast(serviceConfig.value.api_key_set ? 'API Key 已更新' : 'API Key 已清除（无鉴权模式）', 'success')
  } catch (e) {
    showToast('保存失败: ' + e.message, 'error')
  } finally {
    savingServiceConfig.value = false
  }
}

// ---- Quota ----
function isFetchingQuota(id) {
  return fetchingQuotaIds.value.includes(String(id))
}

function setFetchingQuota(id, loading) {
  const key = String(id)
  if (loading) {
    if (!fetchingQuotaIds.value.includes(key)) {
      fetchingQuotaIds.value = [...fetchingQuotaIds.value, key]
    }
    return
  }
  fetchingQuotaIds.value = fetchingQuotaIds.value.filter(item => item !== key)
}

function oauthAccountPriority(account) {
  const code = Number(account?._quota_http_status || 0)
  if (code === 200 && !account?.quota_is_forbidden) return 0
  if (code === 429) return 1
  if (code === 503) return 2
  if (code === 403) return 3
  if (code === 401) return 4
  if (account?.status === 'reauth_required') return 5
  return 6
}

function reorderOAuthAccounts() {
  const decorated = accounts.value.map((account, index) => ({ account, index }))
  decorated.sort((left, right) => {
    const leftIsOAuth = !left.account.account_type || left.account.account_type === 'oauth'
    const rightIsOAuth = !right.account.account_type || right.account.account_type === 'oauth'
    if (leftIsOAuth && rightIsOAuth) {
      const priorityDiff = oauthAccountPriority(left.account) - oauthAccountPriority(right.account)
      if (priorityDiff !== 0) return priorityDiff
    } else if (leftIsOAuth !== rightIsOAuth) {
      return leftIsOAuth ? -1 : 1
    }
    return left.index - right.index
  })
  accounts.value = decorated.map(item => item.account)
}

function applyQuotaResult(result) {
  const acc = accounts.value.find(a => String(a.id) === String(result.id))
  if (!acc) return null

  if (result.success && result.is_forbidden) {
    acc.quota_is_forbidden = true
    acc._quota_http_status = 403
    acc._verified = false
    acc._quota_error = ''
    return 'forbidden'
  }

  if (result.success && (result.quota_5h_used_percent != null || result.quota_7d_used_percent != null || result.total > 0)) {
    acc.quota_is_forbidden = false
    acc._quota_http_status = 200
    acc.quota_5h_used_percent = result.quota_5h_used_percent ?? null
    acc.quota_5h_reset_seconds = result.quota_5h_reset_seconds ?? null
    acc.quota_5h_window_minutes = result.quota_5h_window_minutes ?? null
    acc.quota_7d_used_percent = result.quota_7d_used_percent ?? null
    acc.quota_7d_reset_seconds = result.quota_7d_reset_seconds ?? null
    acc.quota_7d_window_minutes = result.quota_7d_window_minutes ?? null
    acc.quota_total = result.total || null
    acc.quota_used = result.used || null
    acc.quota_reset_at = result.reset || null
    acc.quota_updated_at = new Date().toISOString()
    acc._verified = false
    acc._quota_error = ''
    return 'quota'
  }

  if (result.success && result.verified) {
    acc._quota_http_status = 200
    acc._verified = true
    acc._quota_error = ''
    return 'verified'
  }

  acc._verified = false
  acc._quota_error = result.error || '查询失败'
  acc._quota_http_status = result.http_status || (acc._quota_http_status ?? null)
  // Clear only 5h quota data on failure, keep 7d data for display
  acc.quota_5h_used_percent = null
  acc.quota_5h_reset_seconds = null
  acc.quota_5h_window_minutes = null
  // Keep 7d quota data
  // acc.quota_7d_used_percent = null
  // acc.quota_7d_reset_seconds = null
  // acc.quota_7d_window_minutes = null
  acc.quota_total = null
  acc.quota_used = null
  acc.quota_reset_at = null
  return 'failed'
}

async function fetchQuotaForAccount(account) {
  setFetchingQuota(account.id, true)
  try {
    const res = await longApi.post('/openai/accounts/fetch-quotas', { ids: [account.id] })
    const result = res?.results?.find(r => String(r.id) === String(account.id))
    if (!result) {
      throw new Error('未返回该账号的配额结果')
    }

    const status = applyQuotaResult(result)
    quotaLastFetched.value = new Date().toLocaleTimeString('zh')

    if (status === 'quota') {
      showToast(`${accountDisplayLabel(account) || '账号'} 配额已更新`, 'success')
    } else if (status === 'verified') {
      showToast(`${accountDisplayLabel(account) || '账号'} 账号有效，但未返回配额头`, 'success')
    } else if (status === 'forbidden') {
      showToast(`${accountDisplayLabel(account) || '账号'} 已被禁用`, 'error')
    } else {
      showToast(`${accountDisplayLabel(account) || '账号'} 配额查询失败: ${result.error || '查询失败'}`, 'error')
    }
    reorderOAuthAccounts()
    // Reload accounts to get updated status
    await loadAccounts()
  } catch (e) {
    showToast('配额查询失败: ' + e.message, 'error')
  } finally {
    setFetchingQuota(account.id, false)
  }
}

async function fetchQuotaBatch(ids, scopeLabel = '全部') {
  if (!ids.length) {
    if (oauthAccounts.value.length === 0) {
      showToast(`${scopeLabel}没有OAuth账号，无法查询配额`, 'error')
    } else {
      showToast(`${scopeLabel}没有可查询的账号`, 'error')
    }
    return
  }
  fetchingQuotas.value = true
  try {
    const res = await longApi.post('/openai/accounts/fetch-quotas', { ids })
    let quotaCount = 0
    let verifiedCount = 0
    let failedCount = 0
    let forbiddenCount = 0
    if (res?.results) {
      for (const r of res.results) {
        const status = applyQuotaResult(r)
        if (status === 'forbidden') {
          forbiddenCount++
        } else if (status === 'quota') {
          quotaCount++
        } else if (status === 'verified') {
          verifiedCount++
        } else if (status === 'failed') {
          failedCount++
        }
      }
    }
    quotaLastFetched.value = new Date().toLocaleTimeString('zh')
    const parts = []
    if (quotaCount > 0) parts.push(`${quotaCount} 个有配额数据`)
    if (verifiedCount > 0) parts.push(`${verifiedCount} 个账号有效`)
    if (forbiddenCount > 0) parts.push(`${forbiddenCount} 个被禁用`)
    if (failedCount > 0) parts.push(`${failedCount} 个失败`)
    reorderOAuthAccounts()
    if (parts.length === 0 && failedCount > 0) {
      showToast(`${scopeLabel}查询完成：${failedCount} 个失败`, 'error')
    } else {
      showToast(`${scopeLabel}查询完成：${parts.join('，') || '无返回结果'}`, failedCount > 0 && quotaCount + verifiedCount === 0 ? 'error' : 'success')
    }
    // Reload accounts to get updated status
    await loadAccounts()
  } catch (e) {
    showToast(`${scopeLabel}配额查询失败: ` + e.message, 'error')
  } finally {
    fetchingQuotas.value = false
  }
}

async function fetchAllQuotas() {
  await fetchQuotaBatch(oauthAccounts.value.map(a => a.id), '全部')
}

function openBulkDeleteConfirm() {
  const selectionType = activeTab.value === 'api' ? 'api' : 'oauth'
  const ids = [...getSelectedIds(selectionType)]
  if (!ids.length) {
    showToast('没有可删除的账号', 'error')
    return
  }
  const selectedSet = new Set(ids)
  const list = (selectionType === 'api' ? apiAccounts.value : oauthAccounts.value)
    .filter(a => selectedSet.has(accountId(a.id)))

  bulkDeleteIds.value = ids
  bulkDeleteSelectionType.value = selectionType
  bulkDeleteAccountLabel.value = selectionType === 'api' ? 'API 账号' : 'OAuth 账号'
  bulkDeleteScopeLabel.value = selectionType === 'api'
    ? (allAPISelected.value ? '当前 API 筛选结果' : '手动勾选')
    : (allFilteredOAuthSelected.value ? `当前筛选结果（${quotaFilterLabel.value}）` : '手动勾选')
  bulkDeleteAllSelected.value = selectionType === 'api' ? allAPISelected.value : allFilteredOAuthSelected.value
  bulkDeletePreview.value = list
    .slice(0, 12)
    .map(accountDisplayLabel)
  showBulkDeleteConfirm.value = true
}

function closeBulkDeleteConfirm() {
  if (bulkDeleting.value) return
  resetBulkDeleteConfirm()
}

function resetBulkDeleteConfirm() {
  bulkDeleting.value = false
  showBulkDeleteConfirm.value = false
  bulkDeleteIds.value = []
  bulkDeletePreview.value = []
  bulkDeleteSelectionType.value = 'oauth'
  bulkDeleteAccountLabel.value = 'OAuth 账号'
  bulkDeleteScopeLabel.value = ''
  bulkDeleteAllSelected.value = false
}

async function confirmBulkDelete() {
  const ids = bulkDeleteIds.value
  if (!ids.length) return
  bulkDeleting.value = true
  const deletingTab = bulkDeleteSelectionType.value === 'api' ? 'api' : 'oauth'
  try {
    await api.request({
      method: 'DELETE',
      url: '/openai/accounts',
      data: { ids },
    })
    const idSet = new Set(ids.map(accountId))
    accounts.value = accounts.value.filter(a => !idSet.has(accountId(a.id)))
    if (deletingTab === 'api') {
      selectedAPIIds.value = selectedAPIIds.value.filter(id => !idSet.has(id))
      apiPage.value = 1
    } else {
      selectedOAuthIds.value = selectedOAuthIds.value.filter(id => !idSet.has(id))
      oauthPage.value = 1
    }
    showToast(`已批量删除 ${ids.length} 个账号`, 'success')
    resetBulkDeleteConfirm()
    await loadAccounts()
  } catch (e) {
    showToast('批量删除失败: ' + (e.response?.data?.error || e.message), 'error')
    bulkDeleting.value = false
  }
}

function isTeamPlan(account) {
  return accountPlanType(account) === 'team'
}

function shouldShow5hQuota(account) {
  return !isTeamPlan(account) && account?.quota_5h_used_percent != null
}

function shouldShow7dQuota(account) {
  return account?.quota_7d_used_percent != null
}

function hasDisplayQuotaData(account) {
  return shouldShow5hQuota(account) ||
    shouldShow7dQuota(account) ||
    (account?.quota_total && account.quota_total > 0)
}

function pctBarClass(remainPct) {
  if (remainPct <= 10) return 'bg-red-500'
  if (remainPct <= 30) return 'bg-yellow-500'
  return 'bg-green-500'
}

function shortAccountIdentifier(account) {
  const source = String(account?.chatgpt_account_id || account?.id || '').trim()
  if (!source) return ''
  const compact = source.replace(/[^a-zA-Z0-9]/g, '')
  return compact.slice(-6).toUpperCase()
}

function accountId(id) {
  return String(id)
}

function accountDisplayLabel(account) {
  if (!account) return ''
  if (account.account_type === 'api') {
    const provider = providerDisplayName(account.model_provider)
    const model = account.model ? ` / ${account.model}` : ''
    const baseURL = account.base_url ? ` @ ${account.base_url}` : ''
    return `${provider}${model}${baseURL}`
  }
  if (hideAccountEmails.value) {
    const suffix = shortAccountIdentifier(account)
    return suffix ? `OAuth 账号 #${suffix}` : 'OAuth 账号'
  }
  return account.email || account.chatgpt_account_id || String(account.id)
}

function accountDisplayTitle(account) {
  if (!account) return ''
  if (!account.account_type || account.account_type === 'oauth') {
    return hideAccountEmails.value ? '邮箱已隐藏' : (account.email || account.chatgpt_account_id || '')
  }
  return accountDisplayLabel(account)
}

const scanImportFormatLabels = {
  'easyllm-export': 'EasyLLM',
  'cpa': 'CPA',
  'token': 'Token',
}

function importResultDisplayLabel(result) {
  if (!result) return ''
  const formatTag = result.format ? scanImportFormatLabels[result.format] || result.format : ''
  let base = ''
  if (result.email && hideAccountEmails.value) {
    base = 'OAuth 账号（邮箱已隐藏）'
  } else {
    base = result.email || result.filename || result.token_preview || ''
  }
  if (formatTag && base) return `${base} · ${formatTag}`
  if (formatTag) return formatTag
  return base
}

function getSelectedIds(type) {
  return type === 'api' ? selectedAPIIds.value : selectedOAuthIds.value
}

function setSelectedIds(type, ids) {
  const next = Array.from(new Set(ids.map(accountId)))
  if (type === 'api') {
    selectedAPIIds.value = next
    return
  }
  selectedOAuthIds.value = next
}

function toggleAccountSelection(type, id) {
  const targetId = accountId(id)
  const current = getSelectedIds(type)
  if (current.includes(targetId)) {
    setSelectedIds(type, current.filter(item => item !== targetId))
    return
  }
  setSelectedIds(type, [...current, targetId])
}

function toggleOAuthSelection(id) {
  toggleAccountSelection('oauth', id)
}

function toggleAPISelection(id) {
  toggleAccountSelection('api', id)
}

function isOAuthSelected(id) {
  return selectedOAuthIds.value.includes(accountId(id))
}

function isAPISelected(id) {
  return selectedAPIIds.value.includes(accountId(id))
}

function toggleSelectAllOAuth() {
  const ids = filteredOAuthAccounts.value.map(a => accountId(a.id))
  if (!ids.length) return
  if (allFilteredOAuthSelected.value) {
    const currentSet = new Set(ids)
    setSelectedIds('oauth', selectedOAuthIds.value.filter(id => !currentSet.has(id)))
    return
  }
  setSelectedIds('oauth', [...selectedOAuthIds.value, ...ids])
}

function toggleSelectAllAPI() {
  const ids = filteredAPIAccounts.value.map(a => accountId(a.id))
  if (!ids.length) return
  if (allAPISelected.value) {
    const currentSet = new Set(ids)
    setSelectedIds('api', selectedAPIIds.value.filter(id => !currentSet.has(id)))
    return
  }
  setSelectedIds('api', [...selectedAPIIds.value, ...ids])
}

function clearOAuthSelection() {
  selectedOAuthIds.value = []
}

function clearAPISelection() {
  selectedAPIIds.value = []
}

function pruneSelectedAccountIds() {
  const oauthIdSet = new Set(oauthAccounts.value.map(a => accountId(a.id)))
  const apiIdSet = new Set(apiAccounts.value.map(a => accountId(a.id)))
  selectedOAuthIds.value = selectedOAuthIds.value.filter(id => oauthIdSet.has(id))
  selectedAPIIds.value = selectedAPIIds.value.filter(id => apiIdSet.has(id))
}

function pctColor(remainPct) {
  if (remainPct <= 10) return 'text-red-400'
  if (remainPct <= 30) return 'text-yellow-400'
  return 'text-green-400'
}

function isRegionRestricted(account) {
  const error = String(account?._quota_error || '')
  return Number(account?._quota_http_status) === 403 &&
    (error.includes('unsupported_country_region_territory') || error.includes('Country, region, or territory not supported'))
}

function formatResetTime(seconds) {
  if (!seconds) return ''
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const parts = []
  if (days > 0) parts.push(`${days}d`)
  if (hours > 0) parts.push(`${hours}h`)
  if (minutes > 0 || parts.length === 0) parts.push(`${minutes}m`)
  return parts.join('')
}

// ---- JWT decode ----
function decodeJWTPayload(token) {
  try {
    if (!token || typeof token !== 'string') return null
    const parts = token.split('.')
    if (parts.length !== 3) return null
    const b64 = parts[1].replace(/-/g, '+').replace(/_/g, '/')
    return JSON.parse(atob(b64))
  } catch { return null }
}

function jwtPlanType(account) {
  for (const token of [account?.id_token, account?.access_token]) {
    const payload = decodeJWTPayload(token)
    const plan = String(payload?.['https://api.openai.com/auth']?.chatgpt_plan_type || '').trim().toLowerCase()
    if (plan && plan !== 'free') return plan
  }
  return null
}

const PLAN_LABELS = {
  free:       { text: 'Free',     cls: 'bg-gray-700 text-gray-300' },
  plus:       { text: 'Plus',     cls: 'bg-purple-700/60 text-purple-300' },
  pro:        { text: 'Pro',      cls: 'bg-yellow-700/60 text-yellow-300' },
  prolite:    { text: 'Pro 5x',   cls: 'bg-yellow-700/60 text-yellow-300' },
  promax:     { text: 'Pro 20x',  cls: 'bg-amber-700/60 text-amber-300' },
  team:       { text: 'Team',     cls: 'bg-blue-700/60 text-blue-300' },
  business:   { text: 'Business', cls: 'bg-cyan-700/60 text-cyan-300' },
  enterprise: { text: 'Enterprise', cls: 'bg-emerald-700/60 text-emerald-300' },
}

function accountPlanType(account) {
  const persistedPlan = String(account?.plan || '').trim().toLowerCase()
  return persistedPlan || jwtPlanType(account) || ''
}

function countOAuthAccountsByPlan(plan) {
  return oauthAccounts.value.filter(account => accountPlanType(account) === plan).length
}

function planBadge(account) {
  const plan = accountPlanType(account)
  if (!plan) return null
  return PLAN_LABELS[plan] || { text: plan, cls: 'bg-gray-700 text-gray-400' }
}

// ---- Quota ----
// quotaPct: percentage of quota REMAINING (not used), so green = still available
function quotaPct(account) {
  if (!account.quota_total || account.quota_total <= 0) return 100
  const used = account.quota_used ?? 0
  return Math.max(0, Math.min(100, Math.round((1 - used / account.quota_total) * 100)))
}

function quotaColor(account) {
  const pct = quotaPct(account)   // pct = remaining%
  if (pct <= 10) return 'text-red-400'
  if (pct <= 30) return 'text-yellow-400'
  return 'text-green-400'
}

function quotaBarClass(account) {
  const pct = quotaPct(account)
  if (pct <= 10) return 'bg-red-500'
  if (pct <= 30) return 'bg-yellow-500'
  return 'bg-green-500'
}

function formatQuotaTime(ts) {
  if (!ts) return ''
  const d = new Date(ts)
  const now = new Date()
  const diffMin = Math.round((now - d) / 60000)
  if (diffMin < 1) return '刚刚'
  if (diffMin < 60) return diffMin + '分钟前'
  const diffHr = Math.round(diffMin / 60)
  if (diffHr < 24) return diffHr + '小时前'
  return d.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

// ---- Helpers ----
function maskToken(t) {
  if (!t || t.length < 12) return '***'
  return t.slice(0, 6) + '...' + t.slice(-4)
}

function formatDate(d) {
  if (!d) return ''
  return new Date(d).toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function isExpired(d) {
  return d && new Date(d) < new Date()
}

function isExpiringSoon(d) {
  if (!d) return false
  const diff = new Date(d) - new Date()
  return diff > 0 && diff < 7 * 24 * 60 * 60 * 1000
}

function showToast(message, type = 'success') {
  notify?.(message, type)
}

async function requestOperationConfirm(options) {
  if (typeof confirmOperation !== 'function') return false
  return await confirmOperation(options)
}

watch([filteredOAuthAccounts, filteredAPIAccounts], () => {
  pruneSelectedAccountIds()
  if (oauthPage.value > oauthTotalPages.value) {
    oauthPage.value = oauthTotalPages.value
  }
  if (apiPage.value > apiTotalPages.value) {
    apiPage.value = apiTotalPages.value
  }
})

watch([searchQuery, quotaFilter], () => {
  oauthPage.value = 1
  clearOAuthSelection()
})

watch(apiSearchQuery, () => {
  apiPage.value = 1
  clearAPISelection()
})

onMounted(loadAccounts)
onBeforeUnmount(() => {
  const sessionId = oauthState.value.sessionId
  stopOAuthPolling()
  cancelOAuthSession(sessionId)
})
</script>

<style scoped>
.import-dialog-overlay {
  background: rgba(0, 0, 0, 0.78);
  backdrop-filter: blur(6px);
}
.import-dialog-panel {
  background: #0f172a;
  border-color: rgba(148, 163, 184, 0.28);
  color: #e5e7eb;
}
.import-dialog-panel .border-gray-700 {
  border-color: rgba(148, 163, 184, 0.24) !important;
}
.import-dialog-panel .bg-gray-800,
.import-dialog-panel [class~='bg-gray-800'] {
  background: #1f2937 !important;
}
.import-dialog-panel .text-gray-300 {
  color: #e5e7eb !important;
}
.import-dialog-panel .text-gray-400 {
  color: #cbd5e1 !important;
}
.import-dialog-panel .text-gray-500,
.import-dialog-panel .text-gray-600 {
  color: #94a3b8 !important;
}
.import-dialog-panel .border-gray-600 {
  border-color: rgba(148, 163, 184, 0.58) !important;
}
.import-dialog-panel .border-dashed {
  background: #111827;
}
.import-dialog-panel [class~='bg-green-900/20'] {
  background: #0f2a1c !important;
}
.import-dialog-panel [class~='bg-blue-900/20'] {
  background: #10213f !important;
}
.import-dialog-panel [class~='bg-yellow-900/20'] {
  background: #2d260f !important;
}
.import-dialog-panel [class~='bg-violet-900/20'],
.import-dialog-panel [class~='bg-purple-900/20'] {
  background: #25183f !important;
}
.import-dialog-panel [class~='bg-cyan-900/20'] {
  background: #0d2c35 !important;
}
.import-dialog-panel [class~='text-blue-400/70'] {
  color: #93c5fd !important;
}

.account-card-compact {
  @apply rounded-lg border px-3.5 py-3 transition-all;
  background: var(--app-surface);
  border-color: var(--app-border);
  color: var(--app-text);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
}
.account-card-compact--dense {
  @apply px-2 py-2;
}
.account-card-compact:hover {
  border-color: var(--app-accent-soft);
  box-shadow: 0 10px 26px var(--app-accent-shadow);
}
.account-card-compact--selected {
  border-color: rgba(255, 59, 48, 0.44);
  box-shadow: inset 0 0 0 1px rgba(255, 59, 48, 0.28);
}
.quota-status-badge {
  border: 1px solid transparent;
  letter-spacing: 0.02em;
}
.quota-status-badge--401 {
  background: #c2410c !important;
  border-color: #9a3412 !important;
  color: #fff7ed !important;
}
.quota-status-badge--403 {
  background: #dc2626 !important;
  border-color: #b91c1c !important;
  color: #fef2f2 !important;
}
.quota-status-badge--429 {
  background: #7e22ce !important;
  border-color: #6b21a8 !important;
  color: #faf5ff !important;
}
.quota-status-badge--503,
.quota-status-badge--5xx {
  background: #b45309 !important;
  border-color: #92400e !important;
  color: #fffbeb !important;
}
.quota-status-badge--4xx,
.quota-status-badge--other {
  background: #475569 !important;
  border-color: #334155 !important;
  color: #f8fafc !important;
}
.account-card-compact--api:hover {
  border-color: rgba(52, 199, 89, 0.44);
  box-shadow: 0 10px 26px rgba(52, 199, 89, 0.14);
}

.selection-checkbox {
  @apply relative inline-flex items-center justify-center cursor-pointer;
}
.selection-checkbox input {
  @apply sr-only;
}
.selection-checkbox span {
  @apply relative inline-block w-4 h-4 rounded border transition-colors;
  background: var(--app-control-bg);
  border-color: var(--app-border);
}
.selection-checkbox input:checked + span {
  background: var(--app-danger);
  border-color: var(--app-danger);
}
.selection-checkbox input:checked + span::after {
  content: '';
  position: absolute;
  left: 5px;
  top: 2px;
  width: 4px;
  height: 8px;
  border: solid white;
  border-width: 0 2px 2px 0;
  transform: rotate(45deg);
}

.card-actions {
  @apply flex items-center gap-1.5 overflow-hidden;
}
.card-btn {
  @apply inline-flex h-8 shrink-0 items-center justify-center rounded text-[11px] font-medium transition-colors disabled:opacity-40 whitespace-nowrap;
}
.card-btn--text {
  @apply min-w-[52px] px-2.5;
}
.card-btn--icon {
  @apply h-8 w-8 min-w-8 px-0;
}
.account-card-compact--dense .card-btn {
  @apply h-7 text-[10px];
}
.account-card-compact--dense .card-btn--text {
  @apply min-w-[42px] px-1.5;
}
.account-card-compact--dense .card-btn--icon {
  @apply h-7 w-7 min-w-7 px-0;
}
.dense-account-meta {
  @apply mb-1.5 flex min-h-[20px] items-center gap-1.5 overflow-hidden text-[10px];
  color: var(--app-text-muted);
}
.dense-pill {
  @apply inline-flex shrink-0 items-center rounded px-1.5 py-0.5 text-[9px] font-semibold;
}
.card-btn--primary {
  color: #fff;
  background: var(--app-accent);
}
.card-btn--primary:hover {
  background: var(--app-accent-strong);
}
.card-btn--secondary {
  background: var(--app-control-bg);
  color: var(--app-text-secondary);
  border: 1px solid var(--app-border-soft);
}
.card-btn--secondary:hover {
  background: var(--app-control-hover-bg);
  color: var(--app-text);
}
.card-btn--danger {
  background: transparent;
  color: var(--app-danger);
}
.card-btn--danger:hover {
  background: rgba(255, 59, 48, 0.1);
}
.card-btn--on {
  background: rgba(52, 199, 89, 0.16);
  color: var(--app-success);
}
.card-btn--on:hover {
  background: rgba(52, 199, 89, 0.24);
}
.card-btn--off {
  background: var(--app-control-bg);
  color: var(--app-text-muted);
}
.card-btn--off:hover {
  background: var(--app-control-hover-bg);
  color: var(--app-text-secondary);
}

.stable-actions {
  @apply flex w-full flex-wrap items-center gap-1.5 overflow-visible xl:w-auto xl:justify-end;
}
.account-toolbar {
  @apply grid w-full items-center gap-1.5 overflow-visible;
  grid-template-columns: auto minmax(128px, 1fr) auto auto auto;
}
.account-toolbar--oauth {
  display: grid !important;
  grid-auto-flow: column;
  grid-auto-columns: minmax(72px, 1fr);
  grid-template-columns: none;
  align-items: center;
  gap: 4px;
  width: 100%;
  overflow-x: auto;
  overflow-y: visible;
  white-space: nowrap;
}
.account-toolbar--oauth .toolbar-section {
  display: contents;
}
.account-toolbar--oauth .toolbar-btn {
  @apply min-w-0 px-1.5;
}
.account-toolbar--oauth .toolbar-btn--layout,
.account-toolbar--oauth .toolbar-btn--select {
  @apply min-w-0;
}
.account-toolbar--oauth .toolbar-section--selection {
  @apply justify-start;
}
.account-toolbar--api {
  grid-template-columns: minmax(160px, 1fr) auto auto;
}
.toolbar-section {
  @apply flex min-w-0 items-center gap-1.5;
}
.toolbar-section--actions,
.toolbar-section--view,
.toolbar-section--filter {
  @apply flex-nowrap;
}
.toolbar-section--search {
  @apply min-w-0;
}
.toolbar-section--selection {
  @apply justify-end;
}
.stable-actions > *,
.toolbar-section > * {
  @apply whitespace-nowrap;
}
.toolbar-btn {
  @apply inline-flex h-8 min-w-[50px] shrink-0 items-center justify-center gap-1 rounded-md border px-2 text-[11px] font-medium leading-none transition-colors disabled:cursor-not-allowed disabled:opacity-40 whitespace-nowrap;
}
.toolbar-btn--layout,
.toolbar-btn--select {
  @apply min-w-[62px];
}
.toolbar-btn-neutral {
  background: var(--app-control-bg);
  border-color: var(--app-border);
  color: var(--app-text-secondary);
}
.toolbar-btn-neutral:hover {
  background: var(--app-control-hover-bg);
  color: var(--app-text);
}
.toolbar-btn-success {
  border-color: rgba(52, 199, 89, 0.34);
  background: rgba(52, 199, 89, 0.14);
  color: var(--app-success);
}
.toolbar-btn-success:hover {
  background: rgba(52, 199, 89, 0.22);
}
.toolbar-btn-danger {
  border-color: rgba(255, 59, 48, 0.34);
  background: rgba(255, 59, 48, 0.12);
  color: var(--app-danger);
}
.toolbar-btn-danger:hover {
  background: rgba(255, 59, 48, 0.2);
}
.toolbar-icon-btn {
  @apply inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md border text-xs transition-colors;
  background: var(--app-control-bg);
  border-color: var(--app-border);
  color: var(--app-text-secondary);
}
.toolbar-icon-btn:hover {
  background: var(--app-control-hover-bg);
  color: var(--app-text);
}
.toolbar-input,
.toolbar-select {
  @apply h-8 rounded-md border px-2 text-[11px] transition-colors focus:outline-none;
  background: var(--app-control-bg);
  border-color: var(--app-border);
  color: var(--app-text);
}
.toolbar-input:focus,
.toolbar-select:focus {
  border-color: var(--app-accent);
}
.toolbar-input {
  color: var(--app-text);
}
.toolbar-input::placeholder {
  color: var(--app-text-faint);
}
.toolbar-search {
  @apply w-full min-w-[116px];
}
.account-toolbar--oauth .toolbar-search {
  @apply w-full min-w-0 max-w-none;
}
.toolbar-select {
  @apply w-[86px] min-w-[86px] max-w-[86px] shrink-0 truncate;
}
.toolbar-select--group {
  @apply w-[112px] min-w-[112px] max-w-[160px];
}
.toolbar-select--plan {
  @apply w-[78px] min-w-[78px] max-w-[84px];
}
.toolbar-select--sort {
  @apply w-[96px] min-w-[96px] max-w-[104px];
}
.toolbar-select--quota {
  @apply w-[68px] min-w-[68px] max-w-[68px];
}
.toolbar-status {
  @apply inline-flex h-8 shrink-0 items-center px-1 text-[11px] whitespace-nowrap;
  color: var(--app-text-muted);
}
@media (max-width: 1420px) {
  .account-toolbar--oauth .toolbar-section--filter,
  .account-toolbar--oauth .toolbar-section--selection {
    @apply justify-start;
  }
}
@media (max-width: 1080px) {
  .account-toolbar,
  .account-toolbar--api {
    grid-template-columns: minmax(0, 1fr);
  }
  .toolbar-section {
    @apply flex-wrap justify-start;
  }
  .account-toolbar--oauth .toolbar-section {
    display: contents;
  }
  .toolbar-section--search .toolbar-search {
    @apply max-w-none;
  }
  .account-toolbar--oauth .toolbar-search {
    @apply w-full max-w-none;
  }
}
.account-toolbar--oauth :is(.toolbar-btn, .toolbar-input, .toolbar-select, .toolbar-status) {
  width: 100%;
  min-width: 0;
  max-width: none;
}
.account-toolbar--oauth .toolbar-search {
  min-width: 0;
}
.account-toolbar--oauth .toolbar-select--plan {
  min-width: 0;
}
.account-toolbar--oauth .toolbar-select--quota {
  min-width: 0;
}
.account-toolbar--oauth .toolbar-status {
  justify-content: center;
}
.stable-tabs {
  @apply flex w-full max-w-full gap-1 overflow-x-auto rounded-lg border p-1 sm:w-fit;
  background: var(--app-control-bg);
  border-color: var(--app-border);
}
.stable-tabs > * {
  @apply shrink-0 whitespace-nowrap;
}

.stable-tabs button {
  color: var(--app-text-secondary);
}

.stable-tabs button:hover {
  color: var(--app-text);
}

.stable-tabs button.bg-blue-600 {
  color: #fff !important;
  background: var(--app-accent) !important;
  box-shadow: 0 8px 20px var(--app-accent-shadow);
}

.stable-tabs button .bg-blue-500 {
  background: rgba(255, 255, 255, 0.24) !important;
  color: #fff !important;
}

.stable-tabs button .bg-gray-700 {
  background: var(--app-surface-muted) !important;
  color: var(--app-text-secondary) !important;
}

.btn {
  @apply inline-flex items-center justify-center px-4 py-2 rounded-lg font-medium text-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed whitespace-nowrap;
}
.header-action-btn {
  @apply h-9 min-w-[76px] gap-1.5 px-2.5 py-0 text-sm;
}
.btn-primary {
  color: #fff;
  background: var(--app-accent);
}
.btn-primary:hover {
  background: var(--app-accent-strong);
}
.btn-secondary {
  background: var(--app-control-bg);
  color: var(--app-text);
  border: 1px solid var(--app-border);
}
.btn-secondary:hover {
  background: var(--app-control-hover-bg);
}
.btn-danger {
  color: #fff;
  background: var(--app-danger);
}
.btn-ghost {
  background: transparent;
  color: var(--app-text-secondary);
}
.btn-ghost:hover {
  background: var(--app-control-hover-bg);
  color: var(--app-text);
}
.btn-sm {
  @apply px-2.5 py-1.5 text-xs;
}
.pagination-bar {
  @apply mt-4 flex flex-col items-center justify-between gap-2 text-sm sm:flex-row;
}
.pagination-page-size {
  @apply inline-flex h-8 items-center gap-2 rounded-lg border px-2 text-xs;
  background: var(--app-control-bg);
  border-color: var(--app-border);
  color: var(--app-text-secondary);
}
.pagination-page-size-select {
  @apply h-6 rounded-md border px-1.5 text-xs focus:outline-none;
  background: var(--app-surface);
  border-color: var(--app-border);
  color: var(--app-text);
}
.pagination-page-size-select:focus {
  border-color: var(--app-accent);
}
.pagination-nav {
  @apply flex flex-wrap items-center justify-center gap-2;
}
.input {
  @apply rounded-lg border px-3 py-2 text-sm focus:outline-none;
  background: var(--app-control-bg);
  border-color: var(--app-border);
  color: var(--app-text);
}
.input:focus {
  border-color: var(--app-accent);
}
.preset-btn {
  @apply flex min-w-0 flex-col items-start gap-0.5 rounded-lg border px-3 py-2 text-left transition-colors;
  background: var(--app-control-bg);
  border-color: var(--app-border);
}
.preset-btn:hover {
  background: var(--app-control-hover-bg);
  border-color: var(--app-accent-soft);
}
.preset-btn--active {
  background: var(--app-accent-tint);
  border-color: var(--app-accent);
}

[data-theme-mode='light'] .account-card-compact {
  background: rgba(255, 255, 255, 0.82);
}

[data-theme-mode='light'] .account-card-compact:hover {
  background: rgba(255, 255, 255, 0.96);
}

[data-theme-mode='light'] .toolbar-btn.bg-gray-800,
[data-theme-mode='light'] .toolbar-btn[class~='bg-gray-800'],
[data-theme-mode='light'] .toolbar-btn[class~='bg-blue-600/20'],
[data-theme-mode='light'] .toolbar-btn[class~='bg-sky-600/20'] {
  background: var(--app-control-bg) !important;
  border-color: var(--app-border) !important;
  color: var(--app-text-secondary) !important;
}

[data-theme-mode='light'] .toolbar-btn[class~='bg-blue-600/20'],
[data-theme-mode='light'] .toolbar-btn[class~='bg-sky-600/20'] {
  background: var(--app-accent-tint) !important;
  border-color: var(--app-accent-soft) !important;
  color: var(--app-accent) !important;
}

[data-theme-mode='light'] .account-card-compact .bg-gray-700,
[data-theme-mode='light'] .account-card-compact [class~='bg-gray-700/60'],
[data-theme-mode='light'] .account-card-compact [class~='bg-gray-600/60'] {
  background: var(--app-surface-muted) !important;
}

[data-theme-mode='light'] .account-card-compact .text-blue-300,
[data-theme-mode='light'] .account-card-compact .text-blue-200 {
  color: var(--app-accent) !important;
}
</style>
