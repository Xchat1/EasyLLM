<template>
  <div class="p-6 space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-white">OpenAI / Codex 管理</h1>
        <p class="text-gray-400 text-sm mt-1">管理 OpenAI OAuth 账号及 Codex API 配置</p>
      </div>
      <div class="flex gap-2">
        <button @click="showImportDialog = true" class="btn btn-secondary flex items-center gap-2">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"/>
          </svg>
          批量导入
        </button>
        <button @click="openOAuthDialog" class="btn btn-secondary flex items-center gap-2">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1"/>
          </svg>
          OAuth 登录
        </button>
        <button @click="showAddAPIDialog = true" class="btn btn-primary flex items-center gap-2">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
          </svg>
          添加 API 账号
        </button>
        <button @click="openServiceConfig" class="btn btn-secondary flex items-center gap-2" title="服务配置：代理池开关、对外API Key、调用统计">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
          </svg>
          服务配置
        </button>
      </div>
    </div>

    <!-- Tab bar -->
    <div class="flex gap-1 bg-gray-800 rounded-lg p-1 w-fit">
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
        <div class="flex items-center justify-between mb-3">
          <div class="text-xs text-gray-500">
            <span v-if="quotaLastFetched">配额更新于 {{ quotaLastFetched }}</span>
          </div>
          <div class="flex items-center gap-2 flex-wrap justify-end">
            <button
              @click="refreshAllTokens"
              :disabled="refreshingAllTokens || oauthAccounts.length === 0"
              class="flex items-center gap-1.5 px-3 py-1.5 bg-gray-800 hover:bg-gray-700 border border-gray-700 text-gray-300 rounded-lg text-xs transition-colors disabled:opacity-40"
              title="刷新全部 OAuth 账号的 Token"
            >
              <svg class="w-3.5 h-3.5" :class="refreshingAllTokens ? 'animate-spin' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
              </svg>
              {{ refreshingAllTokens ? '正在刷新所有 Token...' : '刷新所有 Token' }}
            </button>
            <button
              @click="fetchAllQuotas"
              :disabled="fetchingQuotas || oauthAccounts.length === 0"
              class="flex items-center gap-1.5 px-3 py-1.5 bg-gray-800 hover:bg-gray-700 border border-gray-700 text-gray-300 rounded-lg text-xs transition-colors disabled:opacity-40"
              title="查询全部 OAuth 账号配额"
            >
              <svg class="w-3.5 h-3.5" :class="fetchingQuotas ? 'animate-spin' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
              </svg>
              {{ fetchingQuotas ? '正在查询所有配额...' : '查询所有配额' }}
            </button>
            <button
              @click="exportAccounts"
              :disabled="exportingAccounts"
              class="flex items-center gap-1.5 px-3 py-1.5 bg-emerald-700/20 hover:bg-emerald-700/30 border border-emerald-600/40 text-emerald-300 rounded-lg text-xs transition-colors disabled:opacity-40"
              title="导出全部账号的最新落库数据"
            >
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/>
              </svg>
              {{ exportingAccounts ? '正在导出所有最新数据...' : '一键导出所有最新数据' }}
            </button>

            <!-- Search + Quota result filter + bulk select -->
            <input
              v-model="searchQuery"
              class="bg-gray-800 hover:bg-gray-700 border border-gray-700 text-gray-300 rounded-lg text-xs px-2.5 py-1.5 w-40 transition-colors focus:outline-none focus:border-blue-500 placeholder-gray-600"
              placeholder="搜索账号 (email)"
              title="按 email 搜索账号"
            />
            <select
              v-model="quotaFilter"
              class="bg-gray-800 hover:bg-gray-700 border border-gray-700 text-gray-300 rounded-lg text-xs px-2.5 py-1.5 transition-colors focus:outline-none focus:border-blue-500"
              title="按上次配额查询结果过滤账号"
            >
              <option value="all">全部</option>
              <option value="200">200（成功）</option>
              <option value="401">401（失效/未授权）</option>
              <option value="403">403（地区受限/禁止）</option>
              <option value="429">429（限流）</option>
            </select>

            <button
              v-if="filteredOAuthAccounts.length > 0"
              @click="toggleSelectAllOAuth"
              :disabled="bulkDeleting"
              class="flex items-center gap-1.5 px-3 py-1.5 bg-gray-800 hover:bg-gray-700 border border-gray-700 text-gray-300 rounded-lg text-xs transition-colors disabled:opacity-40"
              :title="allFilteredOAuthSelected ? `取消全选当前筛选结果（${filteredOAuthAccounts.length}）` : `全选当前筛选结果（${filteredOAuthAccounts.length}）`"
            >
              {{ allFilteredOAuthSelected ? '取消全选' : `全选(${filteredOAuthAccounts.length})` }}
            </button>
            <button
              v-if="selectedOAuthIds.length > 0"
              @click="clearOAuthSelection"
              :disabled="bulkDeleting"
              class="flex items-center gap-1.5 px-3 py-1.5 bg-gray-800 hover:bg-gray-700 border border-gray-700 text-gray-300 rounded-lg text-xs transition-colors disabled:opacity-40"
              title="清空当前已选 OAuth 账号"
            >
              清空已选
            </button>
            <div v-if="filteredOAuthAccounts.length > 0" class="text-xs text-gray-400 px-1">
              已选 {{ selectedOAuthIds.length }} 个
            </div>
            <button
              v-if="selectedOAuthIds.length > 0"
              @click="openBulkDeleteConfirm"
              :disabled="bulkDeleting"
              class="flex items-center gap-1.5 px-3 py-1.5 bg-red-600/20 hover:bg-red-600/30 border border-red-600/40 text-red-300 rounded-lg text-xs transition-colors disabled:opacity-40"
              :title="`批量删除已选 OAuth 账号（${selectedOAuthIds.length}）`"
            >
              {{ bulkDeleting ? '删除中...' : `批量删除(${selectedOAuthIds.length})` }}
            </button>
          </div>
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5 gap-3">
          <div
            v-for="account in paginatedOAuth"
            :key="account.id"
            class="account-card-compact"
            :class="[
              account.is_codex_active ? 'ring-1 ring-blue-500/60' : '',
              isOAuthSelected(account.id) ? 'account-card-compact--selected' : '',
            ]"
          >
            <!-- Row 1: email + Codex badge -->
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
              <span class="text-sm font-medium text-white truncate flex-1" :title="account.email">{{ account.email }}</span>
              <span v-if="account.is_codex_active" class="shrink-0 text-[10px] font-bold text-blue-300 bg-blue-600/30 px-1.5 py-0.5 rounded">Codex</span>
              <span v-if="account.status === 'reauth_required'" class="shrink-0 text-[10px] font-bold text-red-300 bg-red-600/20 px-1.5 py-0.5 rounded">重登</span>
              <span v-if="account._quota_http_status && account._quota_http_status !== 200" class="shrink-0 text-[10px] font-bold px-1.5 py-0.5 rounded" :class="account._quota_http_status === 403 ? 'text-red-300 bg-red-600/20' : account._quota_http_status === 401 ? 'text-orange-300 bg-orange-600/20' : 'text-yellow-300 bg-yellow-600/20'">{{ account._quota_http_status }}</span>
            </div>
            <!-- Row 2: info + quota -->
            <div class="flex items-center gap-3 text-[11px] text-gray-500 mb-1.5 min-w-0">
              <span v-if="account.expires_at" class="truncate"
                :class="isExpired(account.expires_at) ? 'text-red-400' : isExpiringSoon(account.expires_at) ? 'text-yellow-400' : ''">
                {{ formatDate(account.expires_at) }}
                <span v-if="isExpired(account.expires_at)" class="text-red-400 ml-0.5">过期</span>
              </span>
              <span v-if="account.chatgpt_account_id" class="truncate font-mono">{{ account.chatgpt_account_id.slice(0, 12) }}</span>
            </div>
            <!-- Row 2.5: plan badge + quota bars -->
            <div class="mb-2 space-y-1">
              <!-- Plan type from JWT -->
              <div class="flex items-center gap-2">
                <span
                  v-if="planBadge(account)"
                  class="text-[9px] font-bold px-1.5 py-0.5 rounded uppercase tracking-wide"
                  :class="planBadge(account).cls"
                >{{ planBadge(account).text }}</span>
                <span v-if="isRegionRestricted(account)" class="text-[10px] text-amber-300">地区受限</span>
                <span v-if="account._verified && !hasQuotaData(account)" class="text-[10px] text-green-400">✓ 有效</span>
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
              <template v-if="!account.quota_is_forbidden && hasWindowQuotaData(account)">
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
              <template v-if="!account.quota_is_forbidden && hasWindowQuotaData(account)">
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
              <div v-if="account.quota_updated_at && hasQuotaData(account)" class="text-[9px] text-gray-600 text-right">
                {{ formatQuotaTime(account.quota_updated_at) }}
              </div>

              <!-- No quota data yet -->
              <div v-if="!hasQuotaData(account) && !account.quota_is_forbidden" class="text-[9px] text-gray-600">
                <span v-if="jwtPlanType(account) === 'free'">免费账号·配额头部不开放</span>
                <span v-else>点击卡片「配额」或上方「查询配额」获取</span>
              </div>
            </div>
            <!-- Row 3: all action buttons in one row -->
            <div class="flex items-center gap-1.5">
              <button
                @click="toggleProxy(account)" :disabled="togglingProxyId === account.id"
                class="card-btn" :class="account.proxy_enabled ? 'card-btn--on' : 'card-btn--off'"
                :title="account.proxy_enabled ? '移出代理池' : '加入代理池'"
              >{{ togglingProxyId === account.id ? '...' : account.proxy_enabled ? '代理' : '代理' }}</button>
              <button
                @click="switchAccount(account)" :disabled="switchingId === account.id"
                class="card-btn card-btn--primary"
                title="切换到该账号"
              >{{ switchingId === account.id ? '...' : '切换' }}</button>
              <button
                @click="refreshToken(account)" :disabled="refreshingId === account.id"
                class="card-btn card-btn--secondary" title="刷新 Token"
              >
                <svg class="w-3 h-3" :class="refreshingId === account.id ? 'animate-spin' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
                </svg>
              </button>
              <button
                @click="fetchQuotaForAccount(account)"
                :disabled="fetchingQuotas || isFetchingQuota(account.id)"
                class="card-btn card-btn--secondary"
                title="查询配额"
              >{{ isFetchingQuota(account.id) ? '...' : '配额' }}</button>
              <button @click="deleteAccount(account.id)" class="card-btn card-btn--danger" title="删除">
                <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                </svg>
              </button>
            </div>
          </div>
        </div>
        <!-- Pagination -->
        <div v-if="oauthTotalPages > 1" class="flex items-center justify-center gap-2 mt-4 text-sm">
          <button @click="oauthPage = Math.max(1, oauthPage - 1)" :disabled="oauthPage <= 1" class="btn btn-sm btn-secondary">上一页</button>
          <span class="text-gray-400">{{ oauthPage }} / {{ oauthTotalPages }}<span class="text-gray-600 ml-2">(共 {{ oauthAccounts.length }})</span></span>
          <button @click="oauthPage = Math.min(oauthTotalPages, oauthPage + 1)" :disabled="oauthPage >= oauthTotalPages" class="btn btn-sm btn-secondary">下一页</button>
        </div>
      </template>
    </div>

    <!-- API Accounts Tab -->
    <div v-if="activeTab === 'api'">
      <div v-if="apiAccounts.length === 0" class="text-center py-12 text-gray-500">
        <p class="text-base mb-1">暂无 API 账号</p>
        <p class="text-sm">点击"添加 API 账号"配置自定义 API 端点</p>
      </div>
      <template v-else>
        <div class="flex items-center justify-end gap-2 flex-wrap mb-3">
          <button
            @click="toggleSelectAllAPI"
            :disabled="bulkDeleting"
            class="flex items-center gap-1.5 px-3 py-1.5 bg-gray-800 hover:bg-gray-700 border border-gray-700 text-gray-300 rounded-lg text-xs transition-colors disabled:opacity-40"
            :title="allAPISelected ? `取消全选全部 API 账号（${apiAccounts.length}）` : `全选全部 API 账号（${apiAccounts.length}）`"
          >
            {{ allAPISelected ? '取消全选' : `全选(${apiAccounts.length})` }}
          </button>
          <button
            v-if="selectedAPIIds.length > 0"
            @click="clearAPISelection"
            :disabled="bulkDeleting"
            class="flex items-center gap-1.5 px-3 py-1.5 bg-gray-800 hover:bg-gray-700 border border-gray-700 text-gray-300 rounded-lg text-xs transition-colors disabled:opacity-40"
            title="清空当前已选 API 账号"
          >
            清空已选
          </button>
          <div class="text-xs text-gray-400 px-1">
            已选 {{ selectedAPIIds.length }} 个
          </div>
          <button
            v-if="selectedAPIIds.length > 0"
            @click="openBulkDeleteConfirm"
            :disabled="bulkDeleting"
            class="flex items-center gap-1.5 px-3 py-1.5 bg-red-600/20 hover:bg-red-600/30 border border-red-600/40 text-red-300 rounded-lg text-xs transition-colors disabled:opacity-40"
            :title="`批量删除已选 API 账号（${selectedAPIIds.length}）`"
          >
            {{ bulkDeleting ? '删除中...' : `批量删除(${selectedAPIIds.length})` }}
          </button>
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5 gap-3">
          <div
            v-for="account in paginatedAPI"
            :key="account.id"
            class="account-card-compact account-card-compact--api"
            :class="isAPISelected(account.id) ? 'account-card-compact--selected' : ''"
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
              <span class="text-sm font-medium text-white truncate flex-1" :title="account.model_provider">{{ account.model_provider || 'API' }}</span>
              <span v-if="account.model" class="shrink-0 text-[10px] font-mono text-emerald-300 bg-emerald-600/20 px-1.5 py-0.5 rounded truncate max-w-[100px]">{{ account.model }}</span>
            </div>
            <!-- Row 2: info -->
            <div class="flex items-center gap-3 text-[11px] text-gray-500 mb-2.5 truncate">
              <span v-if="account.base_url" class="truncate font-mono">{{ account.base_url }}</span>
              <span v-if="account.wire_api" class="shrink-0">{{ account.wire_api }}</span>
            </div>
            <!-- Row 3: action buttons -->
            <div class="flex items-center gap-1.5">
              <button @click="switchAPIAccount(account)" :disabled="switchingId === account.id" class="card-btn card-btn--primary flex-1" title="切换配置">
                {{ switchingId === account.id ? '...' : '切换' }}
              </button>
              <button @click="editAPIAccount(account)" class="card-btn card-btn--secondary" title="编辑">
                <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
                </svg>
              </button>
              <button @click="deleteAccount(account.id)" class="card-btn card-btn--danger" title="删除">
                <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                </svg>
              </button>
            </div>
          </div>
        </div>
        <!-- Pagination -->
        <div v-if="apiTotalPages > 1" class="flex items-center justify-center gap-2 mt-4 text-sm">
          <button @click="apiPage = Math.max(1, apiPage - 1)" :disabled="apiPage <= 1" class="btn btn-sm btn-secondary">上一页</button>
          <span class="text-gray-400">{{ apiPage }} / {{ apiTotalPages }}<span class="text-gray-600 ml-2">(共 {{ apiAccounts.length }})</span></span>
          <button @click="apiPage = Math.min(apiTotalPages, apiPage + 1)" :disabled="apiPage >= apiTotalPages" class="btn btn-sm btn-secondary">下一页</button>
        </div>
      </template>
    </div>

    <!-- Toast notification -->
    <div v-if="toast.show" class="fixed bottom-6 right-6 z-[100] px-4 py-3 rounded-lg text-sm font-medium shadow-lg transition-all"
      :class="toast.type === 'success' ? 'bg-green-600 text-white' : 'bg-red-600 text-white'">
      {{ toast.message }}
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
      <div class="bg-gray-900 border border-gray-700 rounded-2xl w-full max-w-md shadow-2xl">
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

    <!-- Batch Import Dialog -->
    <div v-if="showImportDialog" class="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4">
      <div class="bg-gray-900 border border-gray-700 rounded-2xl w-full max-w-xl shadow-2xl">
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
          <div class="flex gap-1 bg-gray-800 rounded-lg p-1">
            <button
              v-for="m in importModes"
              :key="m.id"
              @click="importMode = m.id; importResults = null; importFiles = []; importTokens = []"
              class="flex-1 py-1.5 rounded-md text-xs font-medium transition-colors"
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

          <!-- Mode 2: Scan local directory -->
          <div v-if="importMode === 'scan-dir'">
            <div class="bg-blue-900/20 border border-blue-700/40 rounded-lg p-3 text-xs text-blue-300 mb-3">
              <div class="flex items-start justify-between gap-2">
                <div>
                  🗂 扫描服务器本地目录，自动导入所有 JSON 文件（适合大批量，默认扫 <code>./auth</code>）<br/>
                  <span class="text-blue-400/70">目录内每个 JSON 文件都支持单对象、数组和 NDJSON 多账号格式</span>
                </div>
                <button @click="downloadExample('scan-dir')" class="shrink-0 px-2 py-1 bg-blue-800/60 hover:bg-blue-700/80 text-blue-200 rounded text-xs transition-colors whitespace-nowrap">
                  下载示例
                </button>
              </div>
            </div>
            <div>
              <label class="block text-xs text-gray-400 mb-1">目录路径</label>
              <input v-model="scanDir" class="input w-full font-mono text-sm" placeholder="./auth  或  /Users/xxx/tokens"/>
              <p class="text-xs text-gray-600 mt-1">服务器本地绝对路径或相对路径（相对于程序运行目录）</p>
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

          <!-- Mode 5: Import from Sub2API JSON format -->
          <div v-if="importMode === 'sub2api'">
            <div class="bg-blue-900/20 border border-blue-700/40 rounded-lg p-3 text-xs text-blue-300 mb-3">
              🚀 直接导入 Sub2API 格式的 JSON 文件（支持包含 accounts 数组的文件）
            </div>
            <div v-if="!importSub2APIFile">
              <input ref="importSub2APIFileInput" type="file" accept=".json" class="hidden" @change="handleSub2APIFileSelect"/>
              <div
                @click="$refs.importSub2APIFileInput.click()"
                @dragover.prevent="dragging = true"
                @dragleave.prevent="dragging = false"
                @drop.prevent="handleDrop"
                class="border-2 border-dashed border-gray-600 rounded-xl p-8 text-center cursor-pointer hover:border-blue-500 hover:bg-blue-900/10 transition-colors"
                :class="{ 'border-blue-500 bg-blue-900/10': dragging }"
              >
                <svg class="w-10 h-10 mx-auto mb-3 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"/>
                </svg>
                <p class="text-gray-400 text-sm">点击或拖拽 Sub2API 文件到此处</p>
                <p class="text-xs text-gray-600 mt-1">支持包含 accounts 数组的 JSON 格式</p>
              </div>
            </div>
            <div v-else class="space-y-2">
              <div class="flex items-center justify-between">
                <div class="text-sm text-gray-300">
                  已解析：
                  <span class="text-white font-medium">{{ importSub2APIFile.accounts?.length ?? 0 }}</span> 个账号
                </div>
                <button @click="importSub2APIFile = null; importResults = null" class="text-xs text-gray-500 hover:text-red-400">重新选择</button>
              </div>
            </div>
          </div>

          <!-- Mode 4: Re-import from exported backup JSON -->
          <div v-if="importMode === 'from-export'">
            <div class="bg-purple-900/20 border border-purple-700/40 rounded-lg p-3 text-xs text-purple-300 mb-3">
              📦 直接导入由「服务配置 → 导出账号」生成的备份文件（无需任何 OpenAI API 调用，速度最快）
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
                <span class="text-gray-300 truncate flex-1">{{ r.email || r.filename || r.token_preview }}</span>
                <span v-if="r.error && !r.skipped" class="text-red-400 shrink-0 truncate max-w-[160px]">{{ r.error }}</span>
              </div>
            </div>
          </div>
        </div>

        <div class="flex justify-end gap-3 p-6 border-t border-gray-700">
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
      <div class="bg-gray-900 border border-gray-700 rounded-2xl w-full max-w-md shadow-2xl">
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
            <input v-model="apiForm.model_provider" class="input w-full" placeholder="e.g. openai, anthropic"/>
          </div>
          <div>
            <label class="block text-xs text-gray-400 mb-1">Model <span class="text-red-400">*</span></label>
            <input v-model="apiForm.model" class="input w-full" placeholder="e.g. o3, gpt-4o"/>
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
              <div class="text-2xl font-bold text-purple-400">{{ serviceConfig.total_logs }}</div>
              <div class="text-xs text-gray-400 mt-1">历史日志数</div>
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
                <div class="text-xs text-gray-400 mt-0.5"><code class="text-blue-300">/v1/chat/completions</code> 请求在已加入的账号间轮询</div>
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
              <div class="text-xs text-gray-400 mt-0.5">设置后，外部调用 <code class="text-blue-300">/v1/chat/completions</code> 需在 Header 携带 <code class="text-blue-300">Authorization: Bearer &lt;key&gt;</code></div>
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
                <code class="text-green-300 bg-green-500/10 px-1.5 py-0.5 rounded select-all cursor-pointer" :title="serviceConfig.api_key" @click="copyText(serviceConfig.api_key)">{{ serviceConfig.api_key }}</code>
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
import api, { longApi, openaiAPI } from '@/api/index.js'

// State
const accounts = ref([])
const loading = ref(false)
const activeTab = ref('oauth')

// Proxy endpoints
const baseURL = computed(() => `${window.location.protocol}//${window.location.hostname}:${window.location.port || 8022}`)
const proxyEndpoints = [
  { method: 'POST', path: '/v1/chat/completions', desc: '聊天补全（OpenAI 兼容）' },
  { method: 'GET',  path: '/v1/models',           desc: '获取可用模型列表' },
  { method: 'GET',  path: '/pool/status',         desc: '查看代理账号池状态' },
]

function copyText(text) {
  navigator.clipboard.writeText(text).then(() => showToast('已复制', 'success'))
}
const switchingId = ref(null)
const refreshingId = ref(null)
const refreshingAllTokens = ref(false)
const togglingProxyId = ref(null)
const togglingProxyAll = ref(false)
const fetchingQuotas = ref(false)
const fetchingQuotaIds = ref([])
const quotaLastFetched = ref('')
const quotaFilter = ref('all') // all | 200 | 401 | 403 | 429
const searchQuery = ref('') // search by email
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

const quotaFilterLabel = computed(() => {
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
const scanDir = ref('./auth')
const importSub2APIFile = ref(null)
const importSub2APIFileInput = ref(null)
const importMode = ref('token-files') // 'token-files' | 'scan-dir' | 'refresh-tokens' | 'sub2api' | 'from-export'
const importBackupFile = ref(null)  // 从备份导入用的解析后的 JSON 对象
const importBackupFileInput = ref(null)
const importModes = [
  { id: 'token-files',  label: '⚡ Token文件' },
  { id: 'scan-dir',     label: '🗂 扫描目录' },
  { id: 'refresh-tokens', label: '🔄 refresh_token' },
  { id: 'sub2api',      label: '🚀 Sub2API' },
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
  total_logs: 0,
  api_key_set: false,
  api_key_masked: ''
})
const strategies = [
  { id: 'round_robin', label: '轮询' },
  { id: 'random', label: '随机' },
  { id: 'least_used', label: '最少使用' }
]

// Toast
const toast = ref({ show: false, message: '', type: 'success' })

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
const oauthAccounts = computed(() => accounts.value.filter(a => !a.account_type || a.account_type === 'oauth'))
const apiAccounts = computed(() => accounts.value.filter(a => a.account_type === 'api'))
const proxyEnabledCount = computed(() => accounts.value.filter(a => a.proxy_enabled).length)
const proxyAllOn = computed(() => oauthAccounts.value.length > 0 && proxyEnabledCount.value === oauthAccounts.value.length)

const filteredOAuthAccounts = computed(() => {
  let list = oauthAccounts.value
  // Search filter
  const q = searchQuery.value.trim().toLowerCase()
  if (q) {
    list = list.filter(a => (a.email || '').toLowerCase().includes(q) || (a.chatgpt_account_id || '').toLowerCase().includes(q))
  }
  // Quota status filter
  const f = quotaFilter.value
  if (f === 'all') return list
  const want = Number(f)
  return list.filter(a => {
    if (Number(a._quota_http_status) === want) return true
    if (want === 401 && a.status === 'reauth_required') return true
    return false
  })
})
const allFilteredOAuthSelected = computed(() => (
  filteredOAuthAccounts.value.length > 0 &&
  filteredOAuthAccounts.value.every(a => selectedOAuthIds.value.includes(accountId(a.id)))
))
const allAPISelected = computed(() => (
  apiAccounts.value.length > 0 &&
  apiAccounts.value.every(a => selectedAPIIds.value.includes(accountId(a.id)))
))

// Pagination
const PAGE_SIZE = 20
const oauthPage = ref(1)
const apiPage = ref(1)
const oauthTotalPages = computed(() => Math.ceil(filteredOAuthAccounts.value.length / PAGE_SIZE) || 1)
const apiTotalPages = computed(() => Math.ceil(apiAccounts.value.length / PAGE_SIZE) || 1)
const paginatedOAuth = computed(() => {
  const start = (oauthPage.value - 1) * PAGE_SIZE
  return filteredOAuthAccounts.value.slice(start, start + PAGE_SIZE)
})
const paginatedAPI = computed(() => {
  const start = (apiPage.value - 1) * PAGE_SIZE
  return apiAccounts.value.slice(start, start + PAGE_SIZE)
})

const tabs = computed(() => [
  { id: 'oauth', label: 'OAuth 账号', count: oauthAccounts.value.length },
  { id: 'api', label: 'API 账号', count: apiAccounts.value.length }
])

// Methods
async function loadAccounts() {
  loading.value = true
  try {
    // api interceptor returns response.data directly, so res IS the array
    const res = await api.get('/openai/accounts')
    accounts.value = Array.isArray(res) ? res : (res || [])
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
    showToast(`已切换到 ${account.email}，~/.codex/auth.json 已更新`, 'success')
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
    showToast(`已切换到 ${account.email}，~/.codex/config.toml 已更新`, 'success')
  } catch (e) {
    showToast('切换失败: ' + (e.response?.data?.error || e.message), 'error')
  } finally {
    switchingId.value = null
  }
}

async function refreshAllTokens() {
  if (oauthAccounts.value.length === 0) {
    showToast('没有可刷新的 OAuth 账号', 'error')
    return
  }
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
    showToast(`全部刷新完成：${parts.join('，') || '无可用账号'}`, failed > 0 && success === 0 ? 'error' : 'success')
    await loadAccounts()
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
    showToast(`${account.email} token 刷新成功`, 'success')
    await loadAccounts()
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
    showToast(res.proxy_enabled ? `${account.email} 已加入代理池` : `${account.email} 已移出代理池`, 'success')
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
  'scan-dir': {
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

const canRunImport = computed(() => {
  if (importMode.value === 'token-files') return importFiles.value.length > 0
  if (importMode.value === 'scan-dir') return scanDir.value.trim().length > 0
  if (importMode.value === 'refresh-tokens') return importTokens.value.length > 0
  if (importMode.value === 'sub2api') return !!importSub2APIFile.value
  if (importMode.value === 'from-export') return !!importBackupFile.value
  return false
})

const importButtonLabel = computed(() => {
  if (importMode.value === 'token-files') return `导入 ${importFiles.value.length} 个文件`
  if (importMode.value === 'scan-dir') return `扫描并导入目录`
  if (importMode.value === 'refresh-tokens') return `导入 ${importTokens.value.length} 个账号`
  if (importMode.value === 'sub2api') {
    return `导入 ${importSub2APIFile.value?.accounts?.length ?? 0} 个账号`
  }
  if (importMode.value === 'from-export') {
    const total = (importBackupFile.value?.oauth_accounts?.length ?? 0) + (importBackupFile.value?.api_accounts?.length ?? 0)
    return `从备份导入 ${total} 个账号`
  }
  return '导入'
})

function handleFiles(files) {
  if (!files.length) return
  importFiles.value = Array.from(files)
  importResults.value = null
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
      if (!data.oauth_accounts && !data.api_accounts) {
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

function parseSub2APIFile(file) {
  if (!file) return
  const reader = new FileReader()
  reader.onload = (e) => {
    try {
      const data = JSON.parse(e.target.result)
      if (!data.accounts || !Array.isArray(data.accounts)) {
        showToast('文件格式不正确，缺少 accounts 数组', 'error')
        return
      }
      importSub2APIFile.value = data
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

  const [file] = files
  if (importMode.value === 'refresh-tokens') {
    parseRefreshTokenFile(file)
    return
  }
  if (importMode.value === 'sub2api') {
    parseSub2APIFile(file)
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

function handleSub2APIFileSelect(event) {
  const file = event.target.files?.[0]
  parseSub2APIFile(file)
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

    } else if (importMode.value === 'scan-dir') {
      res = await api.post('/openai/import/scan-dir', { dir: scanDir.value.trim() })
      importResults.value = {
        success: res?.success ?? 0,
        skipped: res?.skipped ?? 0,
        failed:  res?.failed  ?? 0,
        total:   res?.total   ?? 0,
        results: res?.results ?? []
      }

    } else if (importMode.value === 'from-export') {
      // 直接消费备份 JSON，无需 API 调用
      res = await api.post('/openai/import/from-export', {
        oauth_accounts: importBackupFile.value.oauth_accounts || [],
        api_accounts:   importBackupFile.value.api_accounts   || [],
      })
      importResults.value = {
        success: res?.success ?? 0,
        skipped: res?.skipped ?? 0,
        failed:  res?.failed  ?? 0,
        total:   res?.total   ?? 0,
        results: res?.results ?? []
      }

    } else if (importMode.value === 'sub2api') {
      res = await api.post('/openai/import/sub2api', importSub2APIFile.value)
      importResults.value = {
        success: res?.success ?? 0,
        skipped: res?.skipped ?? 0,
        failed:  res?.failed  ?? 0,
        total:   res?.total   ?? 0,
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

    if ((importResults.value?.success ?? 0) > 0) {
      await loadAccounts()
      showToast(`成功导入 ${importResults.value.success} 个账号`, 'success')
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
  importSub2APIFile.value = null
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
    await openaiAPI.exchangeOAuthCode(payload)
    await loadAccounts()
    showOAuthDialog.value = false
    resetOAuthState()
    showToast('OAuth 登录成功', 'success')
  } catch (e) {
    oauthState.value.error = e.message
  } finally {
    oauthState.value.loading = false
  }
}

// ---- API Account ----
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
  apiForm.value = { model_provider: '', model: '', base_url: '', api_key: '', wire_api: 'responses', model_reasoning_effort: '' }
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



async function openServiceConfig() {
  showServiceConfigDialog.value = true
  await loadServiceConfig()
}

async function exportAccounts() {
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
    showToast(`已导出 ${payload.oauth_accounts?.length ?? 0} 个 OAuth + ${payload.api_accounts?.length ?? 0} 个 API 账号（基于后端最新落库数据）`, 'success')
  } catch (e) {
    showToast('导出失败: ' + e.message, 'error')
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
  if (code === 403) return 2
  if (code === 401) return 3
  if (account?.status === 'reauth_required') return 4
  return 5
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
      showToast(`${account.email || '账号'} 配额已更新`, 'success')
    } else if (status === 'verified') {
      showToast(`${account.email || '账号'} 账号有效，但未返回配额头`, 'success')
    } else if (status === 'forbidden') {
      showToast(`${account.email || '账号'} 已被禁用`, 'error')
    } else {
      showToast(`${account.email || '账号'} 配额查询失败: ${result.error || '查询失败'}`, 'error')
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
    ? (allAPISelected.value ? '全部 API 账号' : '手动勾选')
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

function hasQuotaData(account) {
  return account.quota_5h_used_percent != null ||
    account.quota_7d_used_percent != null ||
    (account.quota_total && account.quota_total > 0)
}

function hasWindowQuotaData(account) {
  return account.quota_5h_used_percent != null || account.quota_7d_used_percent != null
}

function pctBarClass(remainPct) {
  if (remainPct <= 10) return 'bg-red-500'
  if (remainPct <= 30) return 'bg-yellow-500'
  return 'bg-green-500'
}

function accountId(id) {
  return String(id)
}

function accountDisplayLabel(account) {
  if (!account) return ''
  if (account.account_type === 'api') {
    const provider = account.model_provider || 'API'
    const model = account.model ? ` / ${account.model}` : ''
    const baseURL = account.base_url ? ` @ ${account.base_url}` : ''
    return `${provider}${model}${baseURL}`
  }
  return account.email || account.chatgpt_account_id || String(account.id)
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
  const ids = apiAccounts.value.map(a => accountId(a.id))
  if (!ids.length) return
  if (allAPISelected.value) {
    setSelectedIds('api', [])
    return
  }
  setSelectedIds('api', ids)
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
  const token = account.access_token
  const payload = decodeJWTPayload(token)
  if (!payload) return null
  // JWT structure: { "https://api.openai.com/auth": { "chatgpt_plan_type": "free" } }
  return payload?.['https://api.openai.com/auth']?.chatgpt_plan_type || null
}

const PLAN_LABELS = {
  free:  { text: 'Free',  cls: 'bg-gray-700 text-gray-300' },
  plus:  { text: 'Plus',  cls: 'bg-purple-700/60 text-purple-300' },
  pro:   { text: 'Pro',   cls: 'bg-yellow-700/60 text-yellow-300' },
  team:  { text: 'Team',  cls: 'bg-blue-700/60 text-blue-300' },
}

function planBadge(account) {
  const plan = jwtPlanType(account)
  if (!plan) return null
  return PLAN_LABELS[plan.toLowerCase()] || { text: plan, cls: 'bg-gray-700 text-gray-400' }
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
  toast.value = { show: true, message, type }
  setTimeout(() => { toast.value.show = false }, 3500)
}

watch([filteredOAuthAccounts, apiAccounts], () => {
  pruneSelectedAccountIds()
  if (oauthPage.value > oauthTotalPages.value) {
    oauthPage.value = oauthTotalPages.value
  }
  if (apiPage.value > apiTotalPages.value) {
    apiPage.value = apiTotalPages.value
  }
})

onMounted(loadAccounts)
onBeforeUnmount(() => {
  const sessionId = oauthState.value.sessionId
  stopOAuthPolling()
  cancelOAuthSession(sessionId)
})
</script>

<style scoped>
.account-card-compact {
  @apply bg-gray-800/80 border border-gray-700 rounded-lg px-3.5 py-3 transition-all;
}
.account-card-compact:hover {
  @apply border-blue-500/40 shadow-md shadow-blue-500/5;
}
.account-card-compact--selected {
  @apply border-red-500/40 ring-1 ring-red-500/30;
}
.account-card-compact--api:hover {
  @apply border-emerald-500/40 shadow-emerald-500/5;
}

.selection-checkbox {
  @apply relative inline-flex items-center justify-center cursor-pointer;
}
.selection-checkbox input {
  @apply sr-only;
}
.selection-checkbox span {
  @apply relative inline-block w-4 h-4 rounded border border-gray-500 bg-gray-900/80 transition-colors;
}
.selection-checkbox input:checked + span {
  @apply bg-red-500 border-red-400;
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

.card-btn {
  @apply inline-flex items-center justify-center px-2 py-1 rounded text-[11px] font-medium transition-colors disabled:opacity-40;
}
.card-btn--primary {
  @apply bg-blue-600/80 text-blue-100 hover:bg-blue-600;
}
.card-btn--secondary {
  @apply bg-gray-700 text-gray-300 hover:bg-gray-600;
}
.card-btn--danger {
  @apply bg-transparent text-red-400/70 hover:text-red-300 hover:bg-red-500/10;
}
.card-btn--on {
  @apply bg-green-500/20 text-green-300 hover:bg-green-500/30;
}
.card-btn--off {
  @apply bg-gray-700/60 text-gray-500 hover:text-gray-300 hover:bg-gray-700;
}

.btn {
  @apply px-4 py-2 rounded-lg font-medium text-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed;
}
.btn-primary {
  @apply bg-blue-600 hover:bg-blue-700 text-white;
}
.btn-secondary {
  @apply bg-gray-700 hover:bg-gray-600 text-gray-200;
}
.btn-danger {
  @apply bg-red-600 hover:bg-red-700 text-white;
}
.btn-ghost {
  @apply bg-transparent hover:bg-gray-700 text-gray-400 hover:text-white;
}
.btn-sm {
  @apply px-2.5 py-1.5 text-xs;
}
.input {
  @apply bg-gray-800 border border-gray-600 rounded-lg px-3 py-2 text-white text-sm focus:outline-none focus:border-blue-500;
}
</style>
