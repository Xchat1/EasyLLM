<template>
  <div class="p-3.5 sm:p-5 lg:p-6 space-y-6">
    <!-- Header -->
    <div class="flex flex-col gap-3.5 lg:flex-row lg:items-center lg:justify-between">
      <div class="flex items-center gap-3 shrink-0">
        <span class="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-gradient-to-br from-amber-500/20 to-orange-600/30 border border-amber-500/30 text-2xl shadow-lg shadow-orange-950/40">
          🚀
        </span>
        <div class="min-w-0">
          <h1 class="text-xl font-bold text-white tracking-tight">Antigravity 渠道管理</h1>
          <p class="text-gray-400 text-xs mt-0.5">Google 账号管理与配额水位监控</p>
        </div>
      </div>
      <div class="stable-actions flex flex-wrap items-center gap-2 shrink-0">
        <button @click="showImportModal = true" class="btn btn-secondary header-action-btn" title="批量导入账号">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"/>
          </svg>
          导入
        </button>
        <button @click="openOAuthModal" class="btn btn-secondary header-action-btn" title="通过 Google OAuth 授权登录">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1"/>
          </svg>
          OAuth 登录
        </button>
        <button @click="openAddModal" class="btn btn-primary header-action-btn" title="手动添加账号">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
          </svg>
          添加账号
        </button>
        <button @click="refreshAll" :disabled="refreshingAll" class="btn btn-secondary header-action-btn" title="全局刷新所有账号配额">
          <svg class="w-4 h-4" :class="{ 'animate-spin': refreshingAll }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
          </svg>
          刷新配额
        </button>
        <button @click="exportAccounts" class="btn btn-secondary header-action-btn" title="导出账号配置">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
          </svg>
          导出
        </button>
        <button
          @click="togglePrivacyMode"
          class="btn header-action-btn"
          :class="privacyMode ? 'bg-amber-500/20 text-amber-300 border-amber-500/40' : 'btn-secondary'"
          :title="privacyMode ? '隐私模式已开启：点击显示账号与头像' : '点击开启隐私模式：隐藏账号ID、邮箱、名称与头像'"
        >
          <svg v-if="privacyMode" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21"/>
          </svg>
          <svg v-else class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/>
          </svg>
          {{ privacyMode ? '隐私已开启' : '隐私模式' }}
        </button>
      </div>
    </div>

    <!-- Toolbar / Filter -->
    <div class="card p-4">
      <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div class="flex flex-1 flex-wrap items-center gap-2.5">
          <div class="relative flex-1 min-w-[200px] max-w-sm">
            <input
              v-model="searchQuery"
              type="text"
              class="input w-full pl-9"
              placeholder="搜索邮箱或名称..."
            />
            <svg class="w-4 h-4 text-gray-500 absolute left-3 top-2.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
            </svg>
          </div>
          <select v-model="statusFilter" class="input w-36">
            <option value="all">所有状态</option>
            <option value="active">正常 (Active)</option>
            <option value="forbidden">无权限 (403)</option>
            <option value="expired">已失效 (401)</option>
          </select>
          <select v-model="sortOrder" class="input w-36">
            <option value="default">默认排序</option>
            <option value="quota_high">配额从高到低</option>
            <option value="quota_low">配额从低到高</option>
            <option value="email">按邮箱排序</option>
          </select>
        </div>
        <div class="flex items-center gap-3 self-end md:self-center">
          <div class="flex items-center gap-1.5 text-xs text-gray-400">
            <span>自动刷新:</span>
            <select v-model="autoRefreshMinutes" @change="setupAutoRefresh" class="input input-sm py-1 px-2 text-xs w-auto">
              <option :value="0">关闭</option>
              <option :value="1">1 分钟</option>
              <option :value="5">5 分钟</option>
              <option :value="10">10 分钟</option>
            </select>
          </div>
          <!-- 视图模式切换 -->
          <div class="flex items-center rounded-xl border border-amber-500/30 bg-amber-500/10 p-0.5 shrink-0">
            <button
              @click="setViewMode('grid')"
              class="px-2.5 py-1 text-xs font-medium rounded-lg transition-colors flex items-center gap-1.5"
              :class="viewMode === 'grid' ? 'bg-amber-500 text-slate-950 font-bold shadow-sm' : 'text-amber-200/80 hover:text-white'"
              title="卡片网格模式"
            >
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z"/>
              </svg>
              <span>卡片</span>
            </button>
            <button
              @click="setViewMode('compact')"
              class="px-2.5 py-1 text-xs font-medium rounded-lg transition-colors flex items-center gap-1.5"
              :class="viewMode === 'compact' ? 'bg-amber-500 text-slate-950 font-bold shadow-sm' : 'text-amber-200/80 hover:text-white'"
              title="紧凑列表模式"
            >
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"/>
              </svg>
              <span>紧凑</span>
            </button>
          </div>
          <span class="text-xs text-gray-400 font-medium whitespace-nowrap">共 {{ filteredAccounts.length }} 个账号</span>
          <button
            v-if="selectedIds.length > 0"
            @click="deleteSelected"
            class="btn btn-sm btn-danger ml-2"
          >
            批量删除 ({{ selectedIds.length }})
          </button>
        </div>
      </div>
    </div>

    <!-- Account List / Grid -->
    <div v-if="loading" class="text-center py-16 text-gray-400">
      <svg class="w-8 h-8 animate-spin mx-auto text-blue-500 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
      </svg>
      加载 Antigravity 账号中...
    </div>

    <div v-else-if="filteredAccounts.length === 0" class="card py-16 text-center text-gray-500 space-y-3">
      <div class="text-4xl">🚀</div>
      <div class="text-base text-gray-300">暂无符合条件的 Antigravity 账号</div>
      <p class="text-xs text-gray-500 max-w-sm mx-auto">
        点击右上角「OAuth 登录」直接通过 Google 授权接入，或通过「添加账号」导入已有的 Refresh Token。
      </p>
      <div class="pt-2">
        <button @click="openOAuthModal" class="btn btn-primary btn-sm">立即 OAuth 登录</button>
      </div>
    </div>

    <!-- 紧凑列表模式 -->
    <div v-else-if="viewMode === 'compact'" class="rounded-2xl border border-amber-500/30 bg-slate-900/90 shadow-2xl backdrop-blur-xl overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs">
          <thead class="border-b border-amber-500/25 bg-amber-500/15 text-amber-200 font-semibold tracking-wider text-[11px] uppercase">
            <tr>
              <th class="p-3.5 w-10">
                <input
                  type="checkbox"
                  :checked="selectedIds.length === filteredAccounts.length && filteredAccounts.length > 0"
                  @change="toggleSelectAll"
                  class="h-4 w-4 rounded border-amber-500/40 bg-slate-800 text-amber-500 focus:ring-0"
                />
              </th>
              <th class="p-3.5 min-w-[210px]">账号身份</th>
              <th class="p-3.5 min-w-[140px]">Claude (5h)</th>
              <th class="p-3.5 min-w-[140px]">Claude (周)</th>
              <th class="p-3.5 min-w-[140px]">Gemini (5h)</th>
              <th class="p-3.5 min-w-[140px]">Gemini (周)</th>
              <th class="p-3.5 w-24">状态</th>
              <th class="p-3.5 text-right min-w-[210px]">操作</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-800/80 text-slate-200">
            <tr
              v-for="acc in filteredAccounts"
              :key="acc.id"
              class="transition-colors duration-150"
              :class="acc.active ? 'bg-amber-500/15 hover:bg-amber-500/20 border-l-4 border-amber-400 font-medium' : 'bg-transparent hover:bg-slate-800/50'"
            >
              <td class="p-3.5">
                <input
                  type="checkbox"
                  :checked="selectedIds.includes(acc.id)"
                  @change="toggleSelect(acc.id)"
                  class="h-4 w-4 rounded border-slate-600 bg-slate-800 text-amber-500 focus:ring-0"
                />
              </td>

              <td class="p-3.5">
                <div class="flex items-center gap-2.5">
                  <div class="h-8 w-8 rounded-lg bg-gradient-to-br from-amber-500 to-orange-600 flex items-center justify-center text-white font-bold text-xs shrink-0 overflow-hidden shadow-sm border border-amber-400/40">
                    <template v-if="privacyMode">🔒</template>
                    <template v-else>
                      <img v-if="acc.picture" :src="acc.picture" class="h-full w-full object-cover" />
                      <span v-else>{{ (acc.display_name || acc.email || 'A').slice(0, 1).toUpperCase() }}</span>
                    </template>
                  </div>
                  <div class="min-w-0">
                    <div class="flex items-center gap-1.5">
                      <span class="font-semibold text-white truncate max-w-[150px]" :title="privacyMode ? '已隐藏' : acc.email">
                        {{ getAccountDisplayName(acc) }}
                      </span>
                      <span v-if="acc.subscription_tier" class="px-1.5 py-0.2 text-[9px] font-mono rounded bg-blue-500/20 text-blue-300 border border-blue-500/30 uppercase shrink-0">
                        {{ acc.subscription_tier }}
                      </span>
                      <span v-if="acc.active" class="px-1.5 py-0.2 text-[9px] rounded-full bg-emerald-500/25 text-emerald-300 border border-emerald-500/40 shrink-0 font-semibold flex items-center gap-1">
                        <span class="h-1 w-1 rounded-full bg-emerald-400"></span>激活
                      </span>
                    </div>
                    <div v-if="!privacyMode && acc.email && acc.display_name && acc.display_name !== acc.email" class="text-[11px] text-slate-400 truncate max-w-[150px]">
                      {{ acc.email }}
                    </div>
                  </div>
                </div>
              </td>

              <!-- Claude 5h -->
              <td class="p-3.5">
                <div class="space-y-1.5 max-w-[130px]">
                  <div class="flex items-center justify-between text-[11px]">
                    <span :class="getBucketPercentage(acc, 'claude:5h') !== null ? getQuotaColorClass(getBucketPercentage(acc, 'claude:5h')) : 'text-slate-400'" class="font-mono font-bold">
                      {{ getBucketPercentage(acc, 'claude:5h') !== null ? `${getBucketPercentage(acc, 'claude:5h')}%` : '--' }}
                    </span>
                    <span class="text-[10px] text-slate-400 truncate">{{ formatResetTime(getBucketResetTime(acc, 'claude:5h'), getBucketPercentage(acc, 'claude:5h')) }}</span>
                  </div>
                  <div class="h-2 w-full rounded-full bg-slate-800 border border-slate-700/60 overflow-hidden p-0.5">
                    <div
                      class="h-full rounded-full transition-all duration-300"
                      :class="getQuotaBarClass(getBucketPercentage(acc, 'claude:5h'))"
                      :style="{ width: `${getBucketPercentage(acc, 'claude:5h') || 0}%` }"
                    ></div>
                  </div>
                </div>
              </td>

              <!-- Claude 周 -->
              <td class="p-3.5">
                <div class="space-y-1.5 max-w-[130px]">
                  <div class="flex items-center justify-between text-[11px]">
                    <span :class="getBucketPercentage(acc, 'claude:weekly') !== null ? getQuotaColorClass(getBucketPercentage(acc, 'claude:weekly')) : 'text-slate-400'" class="font-mono font-bold">
                      {{ getBucketPercentage(acc, 'claude:weekly') !== null ? `${getBucketPercentage(acc, 'claude:weekly')}%` : '--' }}
                    </span>
                    <span class="text-[10px] text-slate-400 truncate">{{ formatResetTime(getBucketResetTime(acc, 'claude:weekly'), getBucketPercentage(acc, 'claude:weekly')) }}</span>
                  </div>
                  <div class="h-2 w-full rounded-full bg-slate-800 border border-slate-700/60 overflow-hidden p-0.5">
                    <div
                      class="h-full rounded-full transition-all duration-300"
                      :class="getQuotaBarClass(getBucketPercentage(acc, 'claude:weekly'))"
                      :style="{ width: `${getBucketPercentage(acc, 'claude:weekly') || 0}%` }"
                    ></div>
                  </div>
                </div>
              </td>

              <!-- Gemini 5h -->
              <td class="p-3.5">
                <div class="space-y-1.5 max-w-[130px]">
                  <div class="flex items-center justify-between text-[11px]">
                    <span :class="getBucketPercentage(acc, 'gemini:5h') !== null ? getQuotaColorClass(getBucketPercentage(acc, 'gemini:5h')) : 'text-slate-400'" class="font-mono font-bold">
                      {{ getBucketPercentage(acc, 'gemini:5h') !== null ? `${getBucketPercentage(acc, 'gemini:5h')}%` : '--' }}
                    </span>
                    <span class="text-[10px] text-slate-400 truncate">{{ formatResetTime(getBucketResetTime(acc, 'gemini:5h'), getBucketPercentage(acc, 'gemini:5h')) }}</span>
                  </div>
                  <div class="h-2 w-full rounded-full bg-slate-800 border border-slate-700/60 overflow-hidden p-0.5">
                    <div
                      class="h-full rounded-full transition-all duration-300"
                      :class="getQuotaBarClass(getBucketPercentage(acc, 'gemini:5h'))"
                      :style="{ width: `${getBucketPercentage(acc, 'gemini:5h') || 0}%` }"
                    ></div>
                  </div>
                </div>
              </td>

              <!-- Gemini 周 -->
              <td class="p-3.5">
                <div class="space-y-1.5 max-w-[130px]">
                  <div class="flex items-center justify-between text-[11px]">
                    <span :class="getBucketPercentage(acc, 'gemini:weekly') !== null ? getQuotaColorClass(getBucketPercentage(acc, 'gemini:weekly')) : 'text-slate-400'" class="font-mono font-bold">
                      {{ getBucketPercentage(acc, 'gemini:weekly') !== null ? `${getBucketPercentage(acc, 'gemini:weekly')}%` : '--' }}
                    </span>
                    <span class="text-[10px] text-slate-400 truncate">{{ formatResetTime(getBucketResetTime(acc, 'gemini:weekly'), getBucketPercentage(acc, 'gemini:weekly')) }}</span>
                  </div>
                  <div class="h-2 w-full rounded-full bg-slate-800 border border-slate-700/60 overflow-hidden p-0.5">
                    <div
                      class="h-full rounded-full transition-all duration-300"
                      :class="getQuotaBarClass(getBucketPercentage(acc, 'gemini:weekly'))"
                      :style="{ width: `${getBucketPercentage(acc, 'gemini:weekly') || 0}%` }"
                    ></div>
                  </div>
                </div>
              </td>

              <!-- Status -->
              <td class="p-3.5">
                <span v-if="acc.status === 'active'" class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[11px] font-semibold bg-emerald-500/15 text-emerald-300 border border-emerald-500/30">
                  <span class="h-1.5 w-1.5 rounded-full bg-emerald-400"></span> 正常
                </span>
                <span v-else-if="acc.status === 'forbidden'" class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[11px] font-semibold bg-rose-500/15 text-rose-300 border border-rose-500/30">
                  <span class="h-1.5 w-1.5 rounded-full bg-rose-400"></span> 403 受限
                </span>
                <span v-else-if="acc.status === 'expired'" class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[11px] font-semibold bg-amber-500/15 text-amber-300 border border-amber-500/30">
                  <span class="h-1.5 w-1.5 rounded-full bg-amber-400"></span> 401 失效
                </span>
                <span v-else class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[11px] font-medium bg-slate-800 text-slate-300 border border-slate-700">
                  {{ acc.status }}
                </span>
              </td>

              <!-- Actions -->
              <td class="p-3.5 text-right">
                <div class="flex items-center justify-end gap-1">
                  <button
                    @click="activateAccount(acc)"
                    :disabled="acc.active || activatingId === acc.id"
                    class="btn btn-xs"
                    :class="acc.active ? 'border border-emerald-500/30 bg-emerald-500/10 text-emerald-300 font-semibold cursor-default' : 'btn-primary'"
                    :title="acc.active ? '当前生效中' : '设为当前账号并同步到 Antigravity IDE、2.0 与 agy CLI'"
                  >
                    {{ acc.active ? '生效中' : '设为当前' }}
                  </button>
                  <button @click="refreshAccount(acc)" :disabled="refreshingId === acc.id" class="btn btn-xs btn-secondary !px-1.5" title="刷新配额">
                    <svg class="w-3.5 h-3.5" :class="{ 'animate-spin': refreshingId === acc.id }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
                    </svg>
                  </button>
                  <button @click="openWakeupModal(acc)" class="btn btn-xs btn-secondary !px-1.5" title="唤醒连通测试">
                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"/>
                    </svg>
                  </button>
                  <button @click="copyToken(acc)" class="btn btn-xs btn-secondary !px-1.5" title="复制 Token">
                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-1M8 5a2 2 0 002 2h2a2 2 0 002-2M8 5a2 2 0 012-2h2a2 2 0 012 2v3m2 4H10m0 0l3-3m-3 3l3 3"/>
                    </svg>
                  </button>
                  <button @click="openEditModal(acc)" class="btn btn-xs btn-secondary !px-1.5" title="编辑">
                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z"/>
                    </svg>
                  </button>
                  <button @click="deleteAccount(acc)" class="btn btn-xs btn-secondary hover:!text-red-300 !px-1.5" title="删除">
                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                    </svg>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- 卡片网格模式 -->
    <div v-else class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      <div
        v-for="acc in filteredAccounts"
        :key="acc.id"
        class="rounded-2xl border border-slate-700/70 bg-slate-900/90 p-5 space-y-4 transition-all duration-200 shadow-xl relative overflow-hidden backdrop-blur-xl hover:border-amber-500/40"
        :class="{ 'ring-2 ring-amber-500/60 border-amber-500/50 bg-gradient-to-b from-amber-950/25 to-slate-900/90': acc.active }"
      >
        <!-- Card Header -->
        <div class="flex items-center justify-between gap-3">
          <div class="flex items-center gap-3 min-w-0">
            <input
              type="checkbox"
              :checked="selectedIds.includes(acc.id)"
              @change="toggleSelect(acc.id)"
              class="h-4 w-4 rounded border-gray-700 bg-gray-900 text-blue-500 focus:ring-0 shrink-0"
            />
            <div class="relative h-10 w-10 shrink-0 rounded-xl bg-gradient-to-br from-amber-600 to-orange-700 flex items-center justify-center text-white font-semibold text-sm shadow overflow-hidden border border-amber-500/30">
              <template v-if="privacyMode">
                <svg class="w-5 h-5 text-amber-200" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"/>
                </svg>
              </template>
              <template v-else>
                <img v-if="acc.picture" :src="acc.picture" alt="" class="h-full w-full object-cover" />
                <span v-else>{{ (acc.display_name || acc.email || 'A').slice(0, 1).toUpperCase() }}</span>
              </template>
            </div>
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <span class="font-semibold text-white text-sm truncate" :title="privacyMode ? '已隐藏' : acc.email">
                  {{ getAccountDisplayName(acc) }}
                </span>
                <span
                  v-if="acc.subscription_tier"
                  class="px-2 py-0.5 text-[10px] font-semibold rounded-md bg-blue-500/15 text-blue-400 border border-blue-500/25 shrink-0 tracking-wide uppercase font-mono"
                >
                  {{ acc.subscription_tier }}
                </span>
                <span v-if="acc.active" class="px-1.5 py-0.5 text-[10px] font-semibold rounded-md bg-emerald-500/20 text-emerald-400 border border-emerald-500/30 shrink-0">
                  当前激活
                </span>
              </div>
              <div v-if="!privacyMode && acc.email && acc.display_name && acc.display_name !== acc.email" class="text-xs text-gray-400 truncate mt-0.5" :title="acc.email">
                {{ acc.email }}
              </div>
            </div>
          </div>

          <div class="flex items-center gap-1.5 shrink-0">
            <span
              v-if="privacyMode"
              class="px-2 py-0.5 text-[10px] font-medium rounded-full bg-amber-500/10 text-amber-300 border border-amber-500/20 flex items-center gap-1"
              title="已隐藏账号详情"
            >
              🔒 隐私
            </span>
            <span
              v-if="acc.tag_name"
              class="px-2 py-0.5 text-[10px] font-medium rounded text-white"
              :style="{ backgroundColor: acc.tag_color || '#4B5563' }"
            >
              {{ acc.tag_name }}
            </span>
          </div>
        </div>

        <!-- Status warning if any (基本报错一致: 统一大小、统一颜色) -->
        <div v-if="acc.status === 'forbidden'" class="rounded-lg bg-rose-500/10 border border-rose-500/20 p-2 text-xs text-rose-400 flex items-center gap-1.5">
          <span class="shrink-0">⚠️</span>
          <span>403 Forbidden: 账号无访问权限或受限</span>
        </div>
        <div v-else-if="acc.status === 'expired'" class="rounded-lg bg-amber-500/10 border border-amber-500/20 p-2 text-xs text-amber-400 flex items-center gap-1.5">
          <span class="shrink-0">⚠️</span>
          <span>Token 已过期，请点击刷新或重新授权</span>
        </div>
        <div v-else-if="acc.status === 'error'" class="rounded-lg bg-rose-500/10 border border-rose-500/20 p-2 text-xs text-rose-400 flex items-center gap-1.5">
          <span class="shrink-0">⚠️</span>
          <span>配额拉取异常: {{ acc.status_message || '未知错误' }}</span>
        </div>

        <!-- Available Credits (if any) -->
        <div v-if="getAccountCredits(acc).length > 0" class="rounded-xl bg-slate-800/80 border border-slate-700/60 p-2.5 shadow-inner">
          <div class="text-[11px] text-gray-300 mb-1 flex items-center justify-between">
            <span>可用积分 / Credits</span>
            <span class="text-amber-400 font-mono font-bold">{{ getAccountCredits(acc)[0].credit_amount || '-' }}</span>
          </div>
          <div class="text-[10px] text-gray-400 truncate">
            {{ getAccountCredits(acc)[0].credit_type }}
          </div>
        </div>

        <!-- Quota Progress Bars -->
        <div class="space-y-2.5 pt-2 border-t border-slate-800/80">
          <div class="flex items-center justify-between text-xs text-gray-400">
            <span class="font-medium">配额监控</span>
            <span v-if="acc.last_refresh_at" class="text-[11px] text-gray-400">
              更新于 {{ formatTime(acc.last_refresh_at) }}
            </span>
          </div>

          <!-- 4 Core Buckets: Claude 5h, Claude Weekly, Gemini 5h, Gemini Weekly -->
          <div class="grid grid-cols-2 gap-2.5">
            <div
              v-for="bucket in getCoreBuckets(acc)"
              :key="bucket.key"
              class="rounded-xl bg-slate-800/80 border border-slate-700/70 p-3 space-y-1.5 hover:border-amber-500/40 transition-colors shadow-inner"
              :title="getBucketHoverText(bucket)"
            >
              <!-- 行 1: 模型与百分比 -->
              <div class="flex items-center justify-between">
                <span class="text-gray-200 truncate text-xs font-semibold">{{ bucket.label }}</span>
                <span :class="bucket.percentage !== null ? getQuotaColorClass(bucket.percentage) : 'text-gray-500'" class="font-mono text-xs font-bold">
                  {{ bucket.percentage !== null ? `${bucket.percentage}%` : '--' }}
                </span>
              </div>

              <!-- 进度条 -->
              <div class="h-2.5 w-full rounded-full bg-slate-900 border border-slate-700/60 p-0.5 overflow-hidden">
                <div
                  v-if="bucket.percentage !== null"
                  class="h-full rounded-full transition-all duration-300"
                  :class="getQuotaBarClass(bucket.percentage)"
                  :style="{ width: `${Math.min(100, Math.max(0, bucket.percentage))}%` }"
                ></div>
              </div>

              <!-- 行 2: 重置倒计时放在下方 -->
              <div class="flex items-center justify-between text-[10px] text-gray-400 pt-0.5">
                <span class="flex items-center gap-1 truncate">
                  <svg class="w-3 h-3 text-gray-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
                  </svg>
                  <span>{{ formatResetTime(bucket.resetTime, bucket.percentage) }}</span>
                </span>
              </div>
            </div>
          </div>

          <!-- Other Models (Collapsible) -->
          <div v-if="getExtraModels(acc).length > 0">
            <button
              @click="toggleExtraModels(acc.id)"
              class="text-[11px] text-gray-400 hover:text-amber-300 flex items-center gap-1 transition-colors mt-1"
            >
              <span>{{ expandedModels[acc.id] ? '收起其他模型' : `展开其余 ${getExtraModels(acc).length} 个模型` }}</span>
              <span>{{ expandedModels[acc.id] ? '▴' : '▾' }}</span>
            </button>
            <div v-if="expandedModels[acc.id]" class="mt-2 space-y-1.5">
              <div
                v-for="m in getExtraModels(acc)"
                :key="m.name"
                class="flex items-center justify-between text-xs px-2.5 py-1.5 rounded-lg bg-slate-800/80 border border-slate-700/60"
              >
                <span class="text-gray-300 truncate max-w-[150px]" :title="m.name">{{ m.display_name || m.name }}</span>
                <span :class="getQuotaColorClass(m.percentage)" class="font-mono text-xs font-semibold">{{ m.percentage }}%</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Notes if any -->
        <div v-if="acc.notes" class="text-xs text-gray-300 italic bg-slate-800/60 p-2.5 rounded-lg border border-slate-700/60 truncate">
          {{ acc.notes }}
        </div>

        <!-- Card Actions -->
        <div class="flex items-center justify-between pt-3 border-t border-slate-800/80 gap-2">
          <!-- 主操作：设为当前 -->
          <button
            @click="activateAccount(acc)"
            :disabled="acc.active || activatingId === acc.id"
            class="btn btn-sm"
            :class="acc.active ? 'border border-emerald-500/30 bg-emerald-500/10 text-emerald-300 font-semibold cursor-default' : 'btn-primary'"
            :title="acc.active ? '当前生效中' : '设为当前账号并同步到 Antigravity IDE、2.0 与 agy CLI'"
          >
            <span v-if="activatingId === acc.id" class="animate-spin text-xs">⏳</span>
            <span v-else-if="acc.active" class="text-xs">✓</span>
            <span v-else class="text-xs">⚡</span>
            <span>{{ acc.active ? '当前生效' : '设为当前' }}</span>
          </button>

          <!-- 次级操作按钮组：标准高度 32px 图标按钮 -->
          <div class="flex items-center gap-1.5">
            <button
              @click="refreshAccount(acc)"
              :disabled="refreshingId === acc.id"
              class="btn btn-sm btn-secondary !px-2.5"
              title="刷新配额"
            >
              <svg class="w-3.5 h-3.5" :class="{ 'animate-spin': refreshingId === acc.id }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
              </svg>
            </button>
            <button
              @click="openWakeupModal(acc)"
              class="btn btn-sm btn-secondary !px-2.5"
              title="唤醒与连通性测试"
            >
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"/>
              </svg>
            </button>
            <button
              @click="copyToken(acc)"
              class="btn btn-sm btn-secondary !px-2.5"
              title="复制 Refresh Token"
            >
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-1M8 5a2 2 0 002 2h2a2 2 0 002-2M8 5a2 2 0 012-2h2a2 2 0 012 2m0 0h2a2 2 0 012 2v3m2 4H10m0 0l3-3m-3 3l3 3"/>
              </svg>
            </button>
            <button
              @click="openEditModal(acc)"
              class="btn btn-sm btn-secondary !px-2.5"
              title="编辑账号"
            >
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z"/>
              </svg>
            </button>
            <button
              @click="deleteAccount(acc)"
              class="btn btn-sm btn-secondary hover:!bg-red-500/20 hover:!border-red-500/40 hover:!text-red-300 !px-2.5"
              title="删除账号"
            >
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
              </svg>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- ── OAuth Modal ─────────────────────────────────────────────── -->
    <div v-if="oauthModal.open" class="modal-overlay" @click.self="closeOAuthModal">
      <div class="modal-content max-w-lg space-y-4">
        <div class="modal-header">
          <div class="flex items-center gap-2">
            <span class="text-xl">🚀</span>
            <h3 class="font-semibold text-white">Google OAuth 授权 (Antigravity)</h3>
          </div>
          <button @click="closeOAuthModal" class="text-gray-400 hover:text-white">✕</button>
        </div>

        <div class="modal-body space-y-4 text-sm text-gray-300">
          <p class="leading-relaxed">
            点击下方按钮在浏览器中打开 Google 授权页面。授权完成后将自动完成回调并获取账号配额。
          </p>

          <div class="rounded-xl bg-slate-800/80 p-3.5 border border-slate-700/70 space-y-2">
            <div class="text-xs text-gray-400 font-medium">授权地址</div>
            <div class="text-xs font-mono text-amber-300 break-all select-all bg-slate-900/90 border border-slate-700/60 p-2.5 rounded-lg">
              {{ oauthModal.authUrl }}
            </div>
            <div class="flex gap-2 pt-1">
              <a
                :href="oauthModal.authUrl"
                target="_blank"
                rel="noopener noreferrer"
                class="btn btn-primary btn-sm flex-1 text-center"
              >
                在浏览器中打开授权链接 ↗
              </a>
              <button @click="copyText(oauthModal.authUrl)" class="btn btn-secondary btn-sm">复制链接</button>
            </div>
          </div>

          <div class="flex items-center gap-3 p-3 rounded-xl bg-blue-500/10 border border-blue-500/20 text-xs text-blue-300">
            <svg class="w-5 h-5 animate-spin shrink-0 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
            </svg>
            <div>
              <div class="font-medium text-white">等待授权回调中...</div>
              <div class="text-gray-400 mt-0.5">本地回调服务监听于端口 {{ oauthModal.port }}</div>
            </div>
          </div>

          <!-- Manual callback url submission fallback -->
          <div class="pt-2 border-t border-gray-800 space-y-2">
            <div class="text-xs text-gray-400">若浏览器未自动回调，请粘贴回调页面重定向的完整链接：</div>
            <div class="flex gap-2">
              <input
                v-model="oauthModal.callbackInput"
                type="text"
                class="input input-sm flex-1"
                placeholder="http://localhost:.../oauth-callback?code=..."
              />
              <button @click="submitManualCallback" class="btn btn-secondary btn-sm">提交回调</button>
            </div>
          </div>
        </div>

        <div class="modal-footer">
          <button @click="closeOAuthModal" class="btn btn-secondary">取消</button>
        </div>
      </div>
    </div>

    <!-- ── Add / Edit Modal ────────────────────────────────────────── -->
    <div v-if="accountModal.open" class="modal-overlay" @click.self="accountModal.open = false">
      <div class="modal-content max-w-md space-y-4">
        <div class="modal-header">
          <h3 class="font-semibold text-white">{{ accountModal.isEdit ? '编辑账号' : '添加账号' }}</h3>
          <button @click="accountModal.open = false" class="text-gray-400 hover:text-white">✕</button>
        </div>

        <div class="modal-body space-y-3">
          <div>
            <label class="label">邮箱 *</label>
            <input
              v-model="accountModal.form.email"
              type="email"
              class="input"
              placeholder="user@example.com"
              :disabled="accountModal.isEdit"
            />
          </div>

          <div v-if="!accountModal.isEdit">
            <label class="label">Refresh Token / Access Token *</label>
            <textarea
              v-model="accountModal.form.refresh_token"
              class="input h-24 font-mono text-xs"
              placeholder="粘贴 Google OAuth Refresh Token 或 Access Token"
            ></textarea>
          </div>

          <div>
            <label class="label">显示名称 (可选)</label>
            <input v-model="accountModal.form.display_name" type="text" class="input" placeholder="如：主力 Google 账号" />
          </div>

          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="label">标签名</label>
              <input v-model="accountModal.form.tag_name" type="text" class="input" placeholder="如：企业/个人" />
            </div>
            <div>
              <label class="label">标签颜色</label>
              <input v-model="accountModal.form.tag_color" type="color" class="input h-10 p-1" />
            </div>
          </div>

          <div>
            <label class="label">备注 (可选)</label>
            <textarea v-model="accountModal.form.notes" class="input h-16 text-xs" placeholder="记录账号用途或配额周期"></textarea>
          </div>
        </div>

        <div class="modal-footer">
          <button @click="accountModal.open = false" class="btn btn-secondary">取消</button>
          <button @click="saveAccount" class="btn btn-primary">保存</button>
        </div>
      </div>
    </div>

    <!-- ── Import Modal ────────────────────────────────────────────── -->
    <div v-if="showImportModal" class="modal-overlay" @click.self="showImportModal = false">
      <div class="modal-content max-w-lg space-y-4">
        <div class="modal-header">
          <h3 class="font-semibold text-white">批量导入 Antigravity 账号</h3>
          <button @click="showImportModal = false" class="text-gray-400 hover:text-white">✕</button>
        </div>

        <div class="modal-body space-y-3">
          <div class="flex gap-2 border-b border-gray-800 pb-2">
            <button
              @click="importMode = 'json'"
              class="btn btn-xs"
              :class="importMode === 'json' ? 'btn-primary' : 'btn-secondary'"
            >
              JSON 数组导入
            </button>
            <button
              @click="importMode = 'tokens'"
              class="btn btn-xs"
              :class="importMode === 'tokens' ? 'btn-primary' : 'btn-secondary'"
            >
              Refresh Token 列表
            </button>
          </div>

          <div v-if="importMode === 'json'">
            <label class="label">粘贴账号 JSON 数组</label>
            <textarea
              v-model="importContent"
              class="input h-48 font-mono text-xs"
              placeholder='[
  { "email": "user1@example.com", "refresh_token": "1//..." },
  { "email": "user2@example.com", "refresh_token": "1//..." }
]'
            ></textarea>
          </div>

          <div v-else>
            <label class="label">每行一个 Refresh Token</label>
            <textarea
              v-model="importContent"
              class="input h-48 font-mono text-xs"
              placeholder="1//04abc...&#10;1//04xyz..."
            ></textarea>
          </div>
        </div>

        <div class="modal-footer">
          <button @click="showImportModal = false" class="btn btn-secondary">取消</button>
          <button @click="submitImport" class="btn btn-primary" :disabled="importing">
            {{ importing ? '导入并查询配额中...' : '开始导入' }}
          </button>
        </div>
      </div>
    </div>

    <!-- ── Wakeup Modal ────────────────────────────────────────────── -->
    <div v-if="wakeupModal.open" class="modal-overlay" @click.self="wakeupModal.open = false">
      <div class="modal-content max-w-lg space-y-4">
        <div class="modal-header">
          <div class="flex items-center gap-2">
            <span class="text-xl">🔔</span>
            <h3 class="font-semibold text-white">Antigravity 唤醒验证</h3>
          </div>
          <button @click="wakeupModal.open = false" class="text-gray-400 hover:text-white">✕</button>
        </div>

        <div class="modal-body space-y-3">
          <div class="text-xs text-gray-400">
            通过 Antigravity 官方 Language Server / PA 流式接口发送最小对话测试，测试账号可用性并提前触发配额重置周期。
          </div>

          <div class="rounded-xl bg-slate-800/80 p-3 border border-slate-700/70 text-xs space-y-1">
            <div class="text-gray-400">测试账号: <span class="text-white font-mono">{{ privacyMode ? '测试账号（已隐藏）' : wakeupModal.account?.email }}</span></div>
            <div class="text-gray-400">Project ID: <span class="text-amber-400 font-mono">{{ privacyMode ? '****' : (wakeupModal.account?.project_id || '自动检测') }}</span></div>
          </div>

          <div>
            <label class="label">选择模型</label>
            <select v-model="wakeupModal.model" class="input">
              <option value="gemini-2.5-flash">Gemini 2.5 Flash (快速)</option>
              <option value="gemini-2.5-pro">Gemini 2.5 Pro</option>
              <option value="claude-3-5-sonnet">Claude 3.5 Sonnet</option>
              <option value="claude-3-7-sonnet">Claude 3.7 Sonnet</option>
            </select>
          </div>

          <div>
            <label class="label">测试提示词</label>
            <input v-model="wakeupModal.prompt" type="text" class="input" placeholder="ping" />
          </div>

          <!-- Result section -->
          <div v-if="wakeupModal.result" class="rounded-xl bg-slate-800/80 border border-slate-700/70 p-3.5 space-y-2">
            <div class="flex items-center justify-between text-xs border-b border-slate-700/70 pb-1.5">
              <span class="text-emerald-400 font-semibold">✓ 唤醒测试成功</span>
              <span class="text-amber-300 font-mono">{{ wakeupModal.result.duration_ms }} ms</span>
            </div>
            <div class="text-xs text-gray-200 font-mono bg-slate-900/90 border border-slate-700/60 p-2.5 rounded-lg max-h-32 overflow-y-auto">
              {{ wakeupModal.result.reply }}
            </div>
            <div class="text-[10px] text-gray-400 flex items-center justify-between">
              <span>Token 用量: 提示 {{ wakeupModal.result.prompt_tokens || 0 }} / 补全 {{ wakeupModal.result.completion_tokens || 0 }}</span>
              <span v-if="wakeupModal.result.response_id" class="font-mono truncate max-w-[160px]">ID: {{ wakeupModal.result.response_id }}</span>
            </div>
          </div>
        </div>

        <div class="modal-footer">
          <button @click="wakeupModal.open = false" class="btn btn-secondary">关闭</button>
          <button @click="runWakeup" :disabled="wakeupModal.running" class="btn btn-primary">
            {{ wakeupModal.running ? '调用中...' : '立即测试' }}
          </button>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, inject } from 'vue'
import { antigravityAPI } from '@/api'

const notify = inject('notify', (msg, type) => alert(msg))

const accounts = ref([])
const loading = ref(false)
const searchQuery = ref('')
const statusFilter = ref('all')
const sortOrder = ref('default')
const selectedIds = ref([])
const activatingId = ref(null)
const refreshingId = ref(null)
const refreshingAll = ref(false)
const expandedModels = ref({})
const nowTimestamp = ref(Date.now())
const autoRefreshMinutes = ref(0)
let timerId = null
let autoRefreshTimer = null

const privacyMode = ref(localStorage.getItem('easyllm.antigravity.privacyMode') === 'true')
const viewMode = ref(localStorage.getItem('easyllm.antigravity.viewMode') || 'grid')

function setViewMode(mode) {
  viewMode.value = mode
  localStorage.setItem('easyllm.antigravity.viewMode', mode)
}

function getBucketPercentage(acc, key) {
  const b = getCoreBuckets(acc).find(x => x.key === key)
  return b ? b.percentage : null
}

function getBucketResetTime(acc, key) {
  const b = getCoreBuckets(acc).find(x => x.key === key)
  return b ? b.resetTime : ''
}

function toggleSelectAll() {
  if (selectedIds.value.length === filteredAccounts.value.length) {
    selectedIds.value = []
  } else {
    selectedIds.value = filteredAccounts.value.map(a => a.id)
  }
}

function togglePrivacyMode() {
  privacyMode.value = !privacyMode.value
  localStorage.setItem('easyllm.antigravity.privacyMode', privacyMode.value ? 'true' : 'false')
  notify(privacyMode.value ? '隐私模式已开启：隐藏账号ID、邮箱与头像' : '隐私模式已关闭：显示账号信息', 'info')
}

function getAccountDisplayName(acc) {
  if (privacyMode.value) {
    const idx = accounts.value.findIndex(a => a.id === acc.id)
    return `账号 #${idx >= 0 ? idx + 1 : (acc.id || '').slice(0, 4)}`
  }
  return acc.display_name || acc.email || '未命名账号'
}

function getAccountSubText(acc) {
  if (privacyMode.value) {
    return 'ID / 邮箱已隐藏'
  }
  return acc.email
}

function setupAutoRefresh() {
  if (autoRefreshTimer) {
    clearInterval(autoRefreshTimer)
    autoRefreshTimer = null
  }
  if (autoRefreshMinutes.value > 0) {
    autoRefreshTimer = setInterval(() => {
      if (document.hidden || window.__easyllm_hidden) return
      refreshAll()
    }, autoRefreshMinutes.value * 60 * 1000)
  }
}

// Modals
const showImportModal = ref(false)
const importMode = ref('json')
const importContent = ref('')
const importing = ref(false)

const oauthModal = ref({
  open: false,
  loginId: '',
  authUrl: '',
  port: 0,
  callbackInput: '',
})

const accountModal = ref({
  open: false,
  isEdit: false,
  editId: null,
  form: { email: '', refresh_token: '', display_name: '', tag_name: '', tag_color: '#3B82F6', notes: '' },
})

const wakeupModal = ref({
  open: false,
  account: null,
  model: 'gemini-2.5-flash',
  prompt: 'ping',
  running: false,
  result: null,
})

// Load accounts
async function loadAccounts() {
  loading.value = true
  try {
    const res = await antigravityAPI.list()
    accounts.value = Array.isArray(res) ? res : []
  } catch (err) {
    notify('加载 Antigravity 账号失败: ' + err.message, 'error')
  } finally {
    loading.value = false
  }
}

// Filtered and sorted accounts
const filteredAccounts = computed(() => {
  let list = accounts.value.slice()

  if (searchQuery.value.trim()) {
    const q = searchQuery.value.trim().toLowerCase()
    list = list.filter(a =>
      (a.email && a.email.toLowerCase().includes(q)) ||
      (a.display_name && a.display_name.toLowerCase().includes(q)) ||
      (a.notes && a.notes.toLowerCase().includes(q))
    )
  }

  if (statusFilter.value !== 'all') {
    list = list.filter(a => a.status === statusFilter.value)
  }

  if (sortOrder.value === 'quota_high') {
    list.sort((a, b) => getAccountAverageQuota(b) - getAccountAverageQuota(a))
  } else if (sortOrder.value === 'quota_low') {
    list.sort((a, b) => getAccountAverageQuota(a) - getAccountAverageQuota(b))
  } else if (sortOrder.value === 'email') {
    list.sort((a, b) => a.email.localeCompare(b.email))
  } else {
    // Default: active first, then created_at desc
    list.sort((a, b) => {
      if (a.active !== b.active) return a.active ? -1 : 1
      return new Date(b.created_at || 0) - new Date(a.created_at || 0)
    })
  }

  return list
})

function getParsedQuotaModels(acc) {
  if (!acc.quota_models_json) return []
  try {
    const arr = JSON.parse(acc.quota_models_json)
    return Array.isArray(arr) ? arr : []
  } catch {
    return []
  }
}

function getAccountCredits(acc) {
  if (!acc.credits_json) return []
  try {
    const arr = JSON.parse(acc.credits_json)
    return Array.isArray(arr) ? arr : []
  } catch {
    return []
  }
}

function getAccountAverageQuota(acc) {
  const models = getParsedQuotaModels(acc)
  if (models.length === 0) return 0
  const sum = models.reduce((acc, m) => acc + (m.percentage || 0), 0)
  return sum / models.length
}

const BUCKET_DEFINITIONS = [
  { key: 'claude:5h', label: 'Claude (5h)', matches: ['3p-5h', 'claude:5h'] },
  { key: 'claude:weekly', label: 'Claude (周)', matches: ['3p-weekly', 'claude:weekly'] },
  { key: 'gemini:5h', label: 'Gemini (5h)', matches: ['gemini-5h', 'gemini:5h'] },
  { key: 'gemini:weekly', label: 'Gemini (周)', matches: ['gemini-weekly', 'gemini:weekly'] },
]

function getCoreBuckets(acc) {
  const models = getParsedQuotaModels(acc)
  return BUCKET_DEFINITIONS.map(def => {
    const found = models.find(m => def.matches.includes(m.name))
    return {
      key: def.key,
      label: def.label,
      percentage: found ? found.percentage : null,
      resetTime: found?.reset_time || '',
    }
  })
}

function getExtraModels(acc) {
  const models = getParsedQuotaModels(acc)
  const coreNames = BUCKET_DEFINITIONS.flatMap(d => d.matches)
  return models.filter(m => !coreNames.includes(m.name))
}

function toggleExtraModels(id) {
  expandedModels.value[id] = !expandedModels.value[id]
}

function getQuotaColorClass(pct) {
  if (pct === null || pct === undefined) return 'text-gray-500'
  if (pct < 20) return 'text-rose-400 font-bold'
  if (pct < 50) return 'text-amber-400 font-semibold'
  if (pct < 80) return 'text-teal-400'
  return 'text-emerald-400 font-bold'
}

function getQuotaBarClass(pct) {
  if (pct === null || pct === undefined) return 'bg-gray-700'
  if (pct < 20) return 'bg-rose-500 shadow-sm shadow-rose-500/50'
  if (pct < 50) return 'bg-amber-500'
  if (pct < 80) return 'bg-teal-500'
  return 'bg-emerald-500 shadow-sm shadow-emerald-500/50'
}

function getBucketHoverText(bucket) {
  if (bucket.percentage === null || bucket.percentage === undefined) return `${bucket.label}: 未查询到配额数据`
  if (bucket.percentage >= 100) {
    return `${bucket.label}: 额度 100% 满额。Antigravity 采用滑动窗口，未消耗时重置点随时间动态顺延；产生调用后将在 5 小时（或对应周周期）后重置。`
  }
  const time = bucket.resetTime ? formatResetTime(bucket.resetTime, bucket.percentage) : '周期计算中'
  return `${bucket.label}: 当前剩余 ${bucket.percentage}% (预计重置: ${time})`
}

function formatTime(t) {
  if (!t) return ''
  const d = new Date(t)
  return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
}

function formatResetTime(isoStr, percentage) {
  if (percentage !== null && percentage !== undefined && percentage >= 100) {
    return '额度满额'
  }
  if (!isoStr) return '实时周期'
  try {
    const d = new Date(isoStr)
    const diffMs = d.getTime() - nowTimestamp.value
    if (diffMs <= 0) return '已到重置点 (待刷新)'
    const diffMin = Math.round(diffMs / 60000)
    const hours = Math.floor(diffMin / 60)
    const mins = diffMin % 60
    const timeStr = d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
    if (diffMin < 60) return `${diffMin}分钟 (${timeStr})`
    if (hours < 24) {
      return `${hours}h ${mins}m (${timeStr})`
    }
    const days = Math.floor(hours / 24)
    const remainHours = hours % 24
    return `${days}天 ${remainHours}h (${timeStr})`
  } catch {
    return isoStr
  }
}

// Selection
function toggleSelect(id) {
  const idx = selectedIds.value.indexOf(id)
  if (idx === -1) selectedIds.value.push(id)
  else selectedIds.value.splice(idx, 1)
}

// Activate Account
async function activateAccount(acc) {
  activatingId.value = acc.id
  try {
    const res = await antigravityAPI.activate(acc.id)
    notify(res.message || '账号已成功切换', res.sync_warning ? 'warning' : 'success')
    await loadAccounts()
  } catch (err) {
    notify('切换激活失败: ' + err.message, 'error')
  } finally {
    activatingId.value = null
  }
}

// Refresh Account
async function refreshAccount(acc) {
  refreshingId.value = acc.id
  try {
    await antigravityAPI.refresh(acc.id)
    notify('配额刷新成功', 'success')
    await loadAccounts()
  } catch (err) {
    notify('配额刷新失败: ' + err.message, 'error')
  } finally {
    refreshingId.value = null
  }
}

// Refresh All Accounts
async function refreshAll() {
  refreshingAll.value = true
  try {
    const res = await antigravityAPI.refreshAll()
    notify(`全局刷新完成：成功 ${res.updated} 个，失败 ${res.failed} 个`, 'success')
    await loadAccounts()
  } catch (err) {
    notify('全局刷新失败: ' + err.message, 'error')
  } finally {
    refreshingAll.value = false
  }
}

// Delete Account
async function deleteAccount(acc) {
  const name = getAccountDisplayName(acc)
  if (!confirm(`确认删除「${name}」吗？`)) return
  try {
    await antigravityAPI.delete(acc.id)
    notify('删除成功', 'success')
    accounts.value = accounts.value.filter(a => a.id !== acc.id)
  } catch (err) {
    notify('删除失败: ' + err.message, 'error')
  }
}

// Batch Delete
async function deleteSelected() {
  if (!confirm(`确认批量删除选中的 ${selectedIds.value.length} 个账号吗？`)) return
  try {
    await antigravityAPI.deleteMany(selectedIds.value)
    notify('批量删除成功', 'success')
    selectedIds.value = []
    await loadAccounts()
  } catch (err) {
    notify('批量删除失败: ' + err.message, 'error')
  }
}

// Copy Token
function copyToken(acc) {
  const token = acc.refresh_token || acc.access_token || ''
  if (!token) {
    notify('该账号暂无 Token 可复制', 'warning')
    return
  }
  copyText(token)
  notify('Token 已复制到剪贴板', 'success')
}

function copyText(str) {
  navigator.clipboard.writeText(str).catch(() => {})
}

// ── OAuth Flow ─────────────────────────────────────────────────────────────
async function openOAuthModal() {
  try {
    const res = await antigravityAPI.startOAuth()
    oauthModal.value = {
      open: true,
      loginId: res.login_id,
      authUrl: res.auth_url,
      port: res.callback_port,
      callbackInput: '',
    }
    // Poll/wait for OAuth completion in background
    pollOAuthCompletion(res.login_id)
  } catch (err) {
    notify('启动 OAuth 授权失败: ' + err.message, 'error')
  }
}

async function pollOAuthCompletion(loginId) {
  try {
    const res = await antigravityAPI.completeOAuth({ login_id: loginId, timeout_sec: 600 })
    if (res && oauthModal.value.open) {
      notify(`授权成功！已添加账号: ${res.email}`, 'success')
      closeOAuthModal()
      await loadAccounts()
    }
  } catch (err) {
    if (oauthModal.value.open) {
      notify('OAuth 授权未完成: ' + err.message, 'error')
    }
  }
}

async function submitManualCallback() {
  if (!oauthModal.value.callbackInput.trim()) {
    notify('请先粘贴回调 URL', 'warning')
    return
  }
  try {
    await antigravityAPI.submitCallback({
      login_id: oauthModal.value.loginId,
      callback_url: oauthModal.value.callbackInput.trim(),
    })
    notify('回调已提交，正在完成登录...', 'info')
  } catch (err) {
    notify('提交回调失败: ' + err.message, 'error')
  }
}

function closeOAuthModal() {
  if (oauthModal.value.loginId) {
    antigravityAPI.cancelOAuth({ login_id: oauthModal.value.loginId }).catch(() => {})
  }
  oauthModal.value.open = false
}

// ── Add / Edit Account Modal ───────────────────────────────────────────────
function openAddModal() {
  accountModal.value = {
    open: true,
    isEdit: false,
    editId: null,
    form: { email: '', refresh_token: '', display_name: '', tag_name: '', tag_color: '#3B82F6', notes: '' },
  }
}

function openEditModal(acc) {
  accountModal.value = {
    open: true,
    isEdit: true,
    editId: acc.id,
    form: {
      email: acc.email,
      refresh_token: '',
      display_name: acc.display_name || '',
      tag_name: acc.tag_name || '',
      tag_color: acc.tag_color || '#3B82F6',
      notes: acc.notes || '',
    },
  }
}

async function saveAccount() {
  const f = accountModal.value.form
  if (!f.email.trim()) {
    notify('邮箱不能为空', 'warning')
    return
  }
  try {
    if (accountModal.value.isEdit) {
      await antigravityAPI.update(accountModal.value.editId, {
        display_name: f.display_name ? f.display_name.trim() : null,
        tag_name: f.tag_name ? f.tag_name.trim() : null,
        tag_color: f.tag_color || null,
        notes: f.notes ? f.notes.trim() : null,
      })
      notify('保存成功', 'success')
    } else {
      if (!f.refresh_token.trim()) {
        notify('请填写 Refresh Token 或 Access Token', 'warning')
        return
      }
      await antigravityAPI.add({
        email: f.email.trim(),
        refresh_token: f.refresh_token.trim(),
        display_name: f.display_name ? f.display_name.trim() : null,
        tag_name: f.tag_name ? f.tag_name.trim() : null,
        tag_color: f.tag_color || null,
        notes: f.notes ? f.notes.trim() : null,
      })
      notify('添加成功并已获取配额', 'success')
    }
    accountModal.value.open = false
    await loadAccounts()
  } catch (err) {
    notify('保存失败: ' + err.message, 'error')
  }
}

// ── Batch Import ───────────────────────────────────────────────────────────
async function submitImport() {
  const content = importContent.value.trim()
  if (!content) {
    notify('导入内容不能为空', 'warning')
    return
  }
  importing.value = true
  try {
    let payload = []
    if (importMode.value === 'json') {
      payload = JSON.parse(content)
      if (!Array.isArray(payload)) {
        throw new Error('JSON 格式必须为数组')
      }
    } else {
      const lines = content.split('\n').map(l => l.trim()).filter(Boolean)
      payload = lines.map(line => ({ refresh_token: line }))
    }
    const res = await antigravityAPI.importAccounts(payload)
    notify(`成功导入 ${res.imported || 0} 个账号`, 'success')
    showImportModal.value = false
    importContent.value = ''
    await loadAccounts()
  } catch (err) {
    notify('导入失败: ' + err.message, 'error')
  } finally {
    importing.value = false
  }
}

// ── Export ─────────────────────────────────────────────────────────────────
async function exportAccounts() {
  try {
    const res = await antigravityAPI.exportJSON()
    const dataStr = 'data:text/json;charset=utf-8,' + encodeURIComponent(JSON.stringify(res, null, 2))
    const dlAnchorElem = document.createElement('a')
    dlAnchorElem.setAttribute('href', dataStr)
    dlAnchorElem.setAttribute('download', `antigravity-accounts-${Date.now()}.json`)
    dlAnchorElem.click()
    notify('导出成功', 'success')
  } catch (err) {
    notify('导出失败: ' + err.message, 'error')
  }
}

// ── Wakeup Modal ───────────────────────────────────────────────────────────
function openWakeupModal(acc) {
  wakeupModal.value = {
    open: true,
    account: acc,
    model: 'gemini-2.5-flash',
    prompt: 'ping',
    running: false,
    result: null,
  }
}

async function runWakeup() {
  if (!wakeupModal.value.account) return
  wakeupModal.value.running = true
  wakeupModal.value.result = null
  try {
    const res = await antigravityAPI.wakeup(wakeupModal.value.account.id, {
      model: wakeupModal.value.model,
      prompt: wakeupModal.value.prompt,
    })
    wakeupModal.value.result = res
    notify('唤醒成功！', 'success')
  } catch (err) {
    notify('唤醒失败: ' + err.message, 'error')
  } finally {
    wakeupModal.value.running = false
  }
}

let lastAutoRefreshTime = 0

async function checkAndAutoRefresh() {
  const now = Date.now()
  if (now - lastAutoRefreshTime < 60000) return // 防抖：至少间隔 1 分钟

  const needsRefresh = accounts.value.some(acc => {
    if (acc.status === 'forbidden') return false
    if (!acc.last_refresh_at) return true
    const lastTime = new Date(acc.last_refresh_at).getTime()
    // 超过 5 分钟未更新
    if (now - lastTime > 5 * 60 * 1000) return true
    // 或者存在配额不足 100% 且已到预计重置点的桶
    const buckets = getCoreBuckets(acc)
    for (const b of buckets) {
      if (b.percentage !== null && b.percentage < 100 && b.resetTime) {
        if (new Date(b.resetTime).getTime() <= now) return true
      }
    }
    return false
  })

  if (needsRefresh && !refreshingAll.value && !refreshingId.value) {
    lastAutoRefreshTime = now
    try {
      await antigravityAPI.refreshAll()
      await loadAccounts()
    } catch (e) {
      console.warn('Auto refresh failed:', e)
    }
  }
}

function checkDueResetPoints() {
  const now = nowTimestamp.value
  const hasExpired = accounts.value.some(acc => {
    if (acc.status === 'forbidden') return false
    const buckets = getCoreBuckets(acc)
    return buckets.some(b => b.percentage !== null && b.percentage < 100 && b.resetTime && new Date(b.resetTime).getTime() <= now)
  })
  if (hasExpired && !refreshingAll.value && !refreshingId.value && (Date.now() - lastAutoRefreshTime > 30000)) {
    checkAndAutoRefresh()
  }
}

let antigravityVisibilityHandler = null

onMounted(async () => {
  await loadAccounts()
  checkAndAutoRefresh()
  timerId = setInterval(() => {
    if (document.hidden || window.__easyllm_hidden) return
    nowTimestamp.value = Date.now()
    checkDueResetPoints()
  }, 10000)

  antigravityVisibilityHandler = () => {
    if (!document.hidden && !window.__easyllm_hidden) {
      nowTimestamp.value = Date.now()
      checkDueResetPoints()
    }
  }
  document.addEventListener('visibilitychange', antigravityVisibilityHandler)
  window.addEventListener('easyllm-app-visibility', antigravityVisibilityHandler)
})

onUnmounted(() => {
  if (timerId) clearInterval(timerId)
  if (autoRefreshTimer) clearInterval(autoRefreshTimer)
  if (antigravityVisibilityHandler) {
    document.removeEventListener('visibilitychange', antigravityVisibilityHandler)
    window.removeEventListener('easyllm-app-visibility', antigravityVisibilityHandler)
  }
})
</script>
