<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { diaryAPI, type Diary, type DiaryStats } from '$lib/api';
	import { formatMonthLabel, formatWeekday, getCurrentMonthString, getMonthRange, shiftMonth, shiftYear } from '$lib/date';
	import { auth } from '$lib/stores.svelte';

	const initialMonth = getCurrentMonthString();

	let diaries = $state<Diary[]>([]);
	let stats = $state<DiaryStats | null>(null);
	let total = $state(0);
	let page = $state(1);
	const searchPerPage = 50;
	const monthPerPage = 50;
	let listSort = $state<'asc' | 'desc'>('asc');
	let searchInput = $state('');
	let searchQuery = $state('');
	let monthCursor = $state(initialMonth);
	let pendingMonth = $state(initialMonth);
	let draftContent = $state('');
	let isLoading = $state(true);
	let isListRefreshing = $state(false);
	let hasLoadedOnce = $state(false);
	let isSaving = $state(false);
	let saveError = $state('');
	let showDeleteModal = $state(false);
	let diaryToDelete = $state<Diary | null>(null);
	let deleteError = $state('');
	let isDeleting = $state(false);

	onMount(() => {
		void initPage();
	});

	async function initPage() {
		await auth.ensureInitialized();
		if (!auth.isLoggedIn) {
			goto('/');
			return;
		}

		await loadData();
	}

	async function loadData() {
		const isInitialLoad = !hasLoadedOnce;
		if (isInitialLoad) {
			isLoading = true;
		} else {
			isListRefreshing = true;
		}

		try {
			const normalizedSearch = searchQuery.trim();
			const isKeywordSearch = normalizedSearch.length > 0;
			const range = isKeywordSearch ? null : getMonthRange(monthCursor);
			const sort = isKeywordSearch || monthCursor !== currentMonth ? listSort : undefined;
			const perPage = isKeywordSearch ? searchPerPage : monthPerPage;
			const currentPage = isKeywordSearch ? page : 1;

			const [diariesResult, statsResult] = await Promise.all([
				diaryAPI.list({
					page: currentPage,
					per_page: perPage,
					search: normalizedSearch,
					start_date: range?.startDate,
					end_date: range?.endDate,
					sort
				}),
				diaryAPI.stats()
			]);

			searchQuery = normalizedSearch;
			diaries = diariesResult.diaries;
			total = diariesResult.total;
			stats = statsResult;
		} catch (err) {
			console.error('Failed to load data:', err);
			diaries = [];
			total = 0;
		} finally {
			if (isInitialLoad) {
				isLoading = false;
				hasLoadedOnce = true;
			}
			isListRefreshing = false;
		}
	}

	async function handleCreateDiary() {
		const content = draftContent.trim();
		if (!content) {
			saveError = '请输入日记内容';
			return;
		}

		isSaving = true;
		saveError = '';

		try {
			const createdDiary = await diaryAPI.create({ content });
			draftContent = '';
			searchInput = '';
			searchQuery = '';
			monthCursor = createdDiary.create_date.slice(0, 7);
			pendingMonth = monthCursor;
			page = 1;
			await loadData();
		} catch (err) {
			saveError = err instanceof Error ? err.message : '保存失败';
		} finally {
			isSaving = false;
		}
	}

	function handleSearch() {
		const normalizedSearch = searchInput.trim();
		searchInput = normalizedSearch;

		if (!normalizedSearch) {
			resetToCurrentMonth();
			return;
		}

		searchQuery = normalizedSearch;
		listSort = 'asc';
		page = 1;
		void loadData();
	}

	function openMonth(month: string) {
		if (!month) {
			return;
		}
		if (month === monthCursor && !searchQuery) {
			pendingMonth = month;
			return;
		}

		monthCursor = month;
		pendingMonth = month;
		searchInput = '';
		searchQuery = '';
		listSort = 'asc';
		page = 1;
		void loadData();
	}

	function applyPendingMonth() {
		if (!searchQuery && pendingMonth === monthCursor) {
			return;
		}
		openMonth(pendingMonth);
	}

	function resetToCurrentMonth() {
		const currentMonth = getCurrentMonthString();
		monthCursor = currentMonth;
		pendingMonth = currentMonth;
		searchInput = '';
		searchQuery = '';
		listSort = 'asc';
		page = 1;
		void loadData();
	}

	function setListSort(nextSort: 'asc' | 'desc') {
		if (listSort === nextSort) {
			return;
		}

		listSort = nextSort;
		page = 1;
		void loadData();
	}

	function shiftCursor(deltaMonths: number) {
		openMonth(shiftMonth(monthCursor, deltaMonths));
	}

	function shiftPendingMonth(deltaMonths: number) {
		pendingMonth = shiftMonth(pendingMonth, deltaMonths);
	}

	function shiftPendingYear(deltaYears: number) {
		pendingMonth = shiftYear(pendingMonth, deltaYears);
	}

	function openDeleteModal(diary: Diary) {
		diaryToDelete = diary;
		deleteError = '';
		showDeleteModal = true;
	}

	function closeDeleteModal() {
		showDeleteModal = false;
		diaryToDelete = null;
		deleteError = '';
	}

	async function confirmDeleteDiary() {
		if (!diaryToDelete) {
			return;
		}

		isDeleting = true;
		deleteError = '';

		try {
			await diaryAPI.delete(diaryToDelete.id);
			closeDeleteModal();
			if (searchQuery && diaries.length === 1 && page > 1) {
				page -= 1;
			}
			await loadData();
		} catch (err) {
			deleteError = err instanceof Error ? err.message : '删除失败';
		} finally {
			isDeleting = false;
		}
	}

	function getDiaryPreview(content: string) {
		return content.length > 90 ? `${content.slice(0, 90)}...` : content;
	}

	function getFrequencyPercent() {
		if (!stats || stats.total_count === 0 || stats.time_span_days === 0) return '0';
		return ((stats.total_count / stats.time_span_days) * 100).toFixed(1);
	}

	function getPageCount() {
		return Math.max(1, Math.ceil(total / searchPerPage));
	}

	function getMonthParts(value: string) {
		const [year, month] = value.split('-').map(Number);
		return { year, month };
	}

	const currentMonth = $derived(getCurrentMonthString());
	const monthLabel = $derived(formatMonthLabel(monthCursor));
	const pendingMonthLabel = $derived(formatMonthLabel(pendingMonth));
	const pendingMonthParts = $derived(getMonthParts(pendingMonth));
	const isSearchMode = $derived(searchQuery.length > 0);
	const isCurrentMonthView = $derived(!searchQuery && monthCursor === currentMonth);
	const isFilteredMonthView = $derived(!searchQuery && monthCursor !== currentMonth);
	const canAdjustSort = $derived(isSearchMode || monthCursor !== currentMonth);
	const canApplyPendingMonth = $derived(isSearchMode || pendingMonth !== monthCursor);
	const listTitle = $derived(
		searchQuery ? '搜索结果' : monthCursor === currentMonth ? '本月日记' : `${formatMonthLabel(monthCursor)}日记`
	);
	const listDescription = $derived(
		searchQuery
			? `关键词搜索结果当前按${listSort === 'asc' ? '从远到近' : '从近到远'}排列。`
			: monthCursor === currentMonth
				? `${formatMonthLabel(monthCursor)}的日记按日期从近到远排列。`
				: `${formatMonthLabel(monthCursor)}的日记当前按${listSort === 'asc' ? '从远到近' : '从近到远'}排列。`
	);
	const timeSelectionHint = $derived(
		isSearchMode
			? '确认后退出关键词搜索，并切换到这一个月份。'
			: pendingMonth === monthCursor
				? '当前展示时间已与上方选择一致。'
				: '确认后会切换到这一个月份。'
	);
	const sortDescription = $derived(
		isSearchMode ? '关键词搜索结果默认从远到近，可切换为从近到远。' : '指定月份结果默认从远到近，可切换为从近到远。'
	);
	const canSearch = $derived(searchInput.trim().length > 0);
</script>

<div class="diary-page">
	{#if isLoading}
		<div class="loading-container">
			<p class="text-surface-500">加载中...</p>
		</div>
	{:else}
		<section class="overview-card card">
			<div>
				<p class="overview-kicker">Diary Archive</p>
				<h1>日记</h1>
				<p class="overview-text">不积跬步，无以成千里；不积小流，无以成江海。</p>
			</div>
			<div class="overview-stats">
				<div>
					<span>累计</span>
					<strong>{stats?.total_count ?? 0}</strong>
				</div>
				<div>
					<span>连续记录</span>
					<strong>{stats?.max_consecutive_days ?? 0} 天</strong>
				</div>
			</div>
		</section>

		<section class="summary-grid">
			<article class="summary-item card">
				<p>总篇数</p>
				<strong>{stats?.total_count ?? 0}</strong>
			</article>
			<article class="summary-item card">
				<p>时间跨度</p>
				<strong>{stats?.time_span_days ?? 0} 天</strong>
			</article>
			<article class="summary-item card">
				<p>记录频率</p>
				<strong>{getFrequencyPercent()}%</strong>
			</article>
			<article class="summary-item card">
				<p>范围</p>
				<strong>{stats?.start_date ? `${stats.start_date} - ${stats.end_date}` : '暂无'}</strong>
			</article>
		</section>

		<section class="composer-card card">
			<div class="composer-head">
				<div>
					<p class="composer-kicker">Write</p>
					<h2>写一封今天的短信</h2>
					<p class="composer-subtext">不用选日期，系统会自动接在上一条日记之后。</p>
				</div>
				<button type="button" class="btn variant-filled-primary" onclick={handleCreateDiary} disabled={isSaving}>
					{isSaving ? '提交中...' : '确认提交'}
				</button>
			</div>

			{#if saveError}
				<div class="alert variant-soft-error">
					<p>{saveError}</p>
				</div>
			{/if}

			<textarea
				bind:value={draftContent}
				class="composer-textarea"
				maxlength="300"
				placeholder="一般不会超过 300 字，想到什么就写什么。"
				onkeydown={(event) => {
					if ((event.metaKey || event.ctrlKey) && event.key === 'Enter' && !isSaving) {
						void handleCreateDiary();
					}
				}}
			></textarea>
			<div class="composer-foot">
				<span>当前 {draftContent.trim().length} / 300 字</span>
				<span>快捷键：Ctrl/Cmd + Enter</span>
			</div>
		</section>

		{#if isListRefreshing}
			<div class="refresh-indicator">正在更新日记列表...</div>
		{/if}

		<section class="filter-card card">
			<div class="filter-head">
				<div>
					<p class="list-kicker">Browse</p>
					<h2>查找日记</h2>
					<p class="filter-copy">关键词搜索、时间查看和排序都集中在这里，切换逻辑更清晰，也更适合移动端。</p>
				</div>
				<div class="filter-status-group">
					<span class="filter-mode-pill">
						{#if isSearchMode}
							关键词搜索中
						{:else if isCurrentMonthView}
							默认显示本月日记
						{:else}
							按时间查看：{monthLabel}
						{/if}
					</span>
					<span class="filter-sub-pill">排序：{listSort === 'asc' ? '从远到近' : '从近到远'}</span>
				</div>
			</div>

			<div class="filter-grid">
				<article class="filter-panel">
					<div class="filter-panel-head">
						<div>
							<p class="filter-panel-kicker">关键词搜索</p>
							<h3>全局模糊查询</h3>
						</div>
						{#if isSearchMode}
							<span class="filter-panel-badge">跨月份</span>
						{/if}
					</div>

					<label class="filter-search-field">
						<span>输入关键词</span>
						<input
							class="input"
							type="text"
							bind:value={searchInput}
							placeholder="例如：散步、工作总结、心情"
							onkeydown={(event) => event.key === 'Enter' && handleSearch()}
						/>
					</label>

					<p class="filter-helper">支持整句、分词和模糊检索，不会自动叠加月份条件。</p>

					<div class="filter-actions">
						<button type="button" onclick={handleSearch} class="btn variant-filled-primary" disabled={!canSearch}>搜索</button>
						{#if searchInput || isSearchMode}
							<button type="button" onclick={resetToCurrentMonth} class="btn variant-soft-surface">清除并看本月</button>
						{/if}
					</div>
				</article>

				<article class="filter-panel filter-panel-compact">
					<div class="filter-panel-head">
						<div>
							<p class="filter-panel-kicker">排序</p>
							<h3>控制列表时间顺序</h3>
						</div>
						{#if canAdjustSort}
							<span class="filter-panel-badge">已启用</span>
						{:else}
							<span class="filter-panel-badge">本月固定</span>
						{/if}
					</div>

					<p class="filter-helper">{sortDescription}</p>

					<div class="sort-segment" aria-label="日记排序顺序">
						<button
							type="button"
							class:sort-segment-active={listSort === 'asc'}
							class="sort-segment-button"
							onclick={() => setListSort('asc')}
							disabled={!canAdjustSort}
						>
							从远到近
						</button>
						<button
							type="button"
							class:sort-segment-active={listSort === 'desc'}
							class="sort-segment-button"
							onclick={() => setListSort('desc')}
							disabled={!canAdjustSort}
						>
							从近到远
						</button>
					</div>

					<p class="sort-tip">{#if canAdjustSort}切换后会刷新当前筛选结果。{:else}本月日记保持默认从近到远展示。{/if}</p>
				</article>

				<article class="filter-panel">
					<div class="filter-panel-head">
						<div>
							<p class="filter-panel-kicker">按时间查看</p>
							<h3>先选择，再确认切换</h3>
						</div>
						<span class="filter-panel-badge">待切换：{pendingMonthLabel}</span>
					</div>

					<div class="time-selector-card">
						<div class="time-selector-section">
							<span class="time-selector-label">年份</span>
							<div class="time-stepper">
								<button type="button" class="btn btn-sm variant-soft-surface time-stepper-button" onclick={() => shiftPendingYear(-1)} aria-label="前一年">
									-1 年
								</button>
								<div class="time-stepper-value">
									<strong>{pendingMonthParts.year}</strong>
									<small>当前选择</small>
								</div>
								<button type="button" class="btn btn-sm variant-soft-surface time-stepper-button" onclick={() => shiftPendingYear(1)} aria-label="后一年">
									+1 年
								</button>
							</div>
						</div>

						<div class="time-selector-section">
							<span class="time-selector-label">月份</span>
							<div class="time-stepper">
								<button type="button" class="btn btn-sm variant-soft-surface time-stepper-button" onclick={() => shiftPendingMonth(-1)} aria-label="前一个月">
									-1 月
								</button>
								<div class="time-stepper-value">
									<strong>{pendingMonthParts.month} 月</strong>
									<small>{pendingMonthLabel}</small>
								</div>
								<button type="button" class="btn btn-sm variant-soft-surface time-stepper-button" onclick={() => shiftPendingMonth(1)} aria-label="后一个月">
									+1 月
								</button>
							</div>
						</div>
					</div>

					<p class="filter-helper">{timeSelectionHint}</p>

					<div class="filter-actions">
						<button type="button" onclick={applyPendingMonth} class="btn variant-filled-primary" disabled={!canApplyPendingMonth}>确认查看</button>
						<button type="button" onclick={resetToCurrentMonth} class="btn variant-soft-surface" disabled={!isSearchMode && isCurrentMonthView}>
							回到本月
						</button>
					</div>
				</article>
			</div>
		</section>

		{#if isSearchMode}
			<div class="active-filters">
				<span>关键词“{searchQuery}”</span>
				<span>共 {total} 篇</span>
			</div>
		{:else if !isCurrentMonthView}
			<div class="active-filters">
				<span>正在查看 {monthLabel}</span>
				<span>共 {total} 篇</span>
			</div>
		{/if}

		<section class="list-card card">
			<div class="list-card-head">
				<div>
					<p class="list-kicker">Entries</p>
					<h2>{listTitle}</h2>
					<p class="list-copy">{listDescription}</p>
				</div>
				<div class="list-card-meta">
					<span>共 {total} 篇</span>
					{#if isSearchMode || isFilteredMonthView}
						<span>{listSort === 'asc' ? '从远到近' : '从近到远'}</span>
					{/if}
					{#if isSearchMode}
						<span>本页 {diaries.length} 篇</span>
					{:else}
						<span>按日期倒序</span>
					{/if}
				</div>
			</div>

			{#if diaries.length === 0}
				<div class="empty-state">
					{#if isSearchMode}
						<p>没有符合关键词的日记。</p>
					{:else if isCurrentMonthView}
						<p>本月还没有日记。</p>
					{:else}
						<p>{monthLabel}还没有日记。</p>
					{/if}
				</div>
			{:else}
				<div class="diary-list">
					{#each diaries as diary (diary.id)}
						<article class="diary-row">
							<div class="diary-row-meta">
								<p class="diary-date">{diary.create_date}</p>
								<p class="diary-weekday">{formatWeekday(diary.create_date)}</p>
							</div>
							<div class="diary-row-body">
								<p class="diary-content">{diary.content}</p>
								<div class="diary-row-foot">
									<span class="diary-length">约 {diary.content.length} 字</span>
									<div class="diary-row-actions">
										<button type="button" class="btn btn-sm btn-quiet" onclick={() => openDeleteModal(diary)}>删除</button>
									</div>
								</div>
							</div>
						</article>
					{/each}
				</div>

				{#if !isSearchMode}
					<div class="month-pagination-shell">
						<div class="month-pagination-copy">
							<p class="month-pagination-title">继续按时间查看</p>
							<p class="month-pagination-subtitle">用列表底部直接切换到相邻月份，不必回到上方操作区。</p>
						</div>
						<div class="month-pagination-actions">
							<button type="button" class="btn variant-soft-surface" onclick={() => shiftCursor(-1)}>上个月</button>
							<button type="button" class="btn variant-soft-surface" onclick={() => shiftCursor(1)}>下个月</button>
						</div>
					</div>
				{/if}

				{#if isSearchMode && total > searchPerPage}
					<div class="pagination">
						<button
							class="btn variant-soft-surface"
							disabled={page === 1}
							onclick={() => {
								page -= 1;
								void loadData();
							}}
						>
							上一页
						</button>
						<span>{page} / {getPageCount()}</span>
						<button
							class="btn variant-soft-surface"
							disabled={page >= getPageCount()}
							onclick={() => {
								page += 1;
								void loadData();
							}}
						>
							下一页
						</button>
					</div>
				{/if}
			{/if}
		</section>
	{/if}

	{#if showDeleteModal && diaryToDelete}
		<div class="modal-backdrop" role="presentation" onclick={(event) => event.target === event.currentTarget && closeDeleteModal()}>
			<div class="modal-card card delete-modal" role="dialog" aria-modal="true" aria-labelledby="diary-delete-title">
				<div class="modal-head">
					<div>
						<p class="list-kicker">Delete Diary</p>
						<h2 id="diary-delete-title">删除日记</h2>
					</div>
					<button class="btn btn-quiet" type="button" onclick={closeDeleteModal}>关闭</button>
				</div>
				<p class="modal-copy">删除后无法恢复，也不支持编辑。确认要删除这篇日记吗？</p>
				<div class="delete-meta">
					<span>{diaryToDelete.create_date}</span>
					<span>{formatWeekday(diaryToDelete.create_date)}</span>
				</div>
				<p class="delete-preview">{getDiaryPreview(diaryToDelete.content)}</p>
				{#if deleteError}
					<div class="alert variant-soft-error">
						<p>{deleteError}</p>
					</div>
				{/if}
				<div class="modal-actions">
					<button type="button" class="btn variant-soft-surface" onclick={closeDeleteModal}>取消</button>
					<button type="button" class="btn variant-filled-error" onclick={confirmDeleteDiary} disabled={isDeleting}>
						{isDeleting ? '删除中...' : '确认删除'}
					</button>
				</div>
			</div>
		</div>
	{/if}
</div>

<style>
	.diary-page {
		max-width: 1080px;
		margin: 0 auto;
		padding: 12px 0 40px;
		display: grid;
		gap: 18px;
	}

	.loading-container {
		display: flex;
		justify-content: center;
		padding: 72px 0;
	}

	.refresh-indicator {
		position: sticky;
		top: 8px;
		z-index: 2;
		justify-self: end;
		padding: 8px 12px;
		border-radius: 999px;
		background: rgba(255, 248, 240, 0.96);
		border: 1px solid rgba(140, 90, 60, 0.18);
		color: var(--color-ink-soft);
		font-size: 0.88rem;
		box-shadow: var(--shadow-card);
	}

	.overview-card {
		display: flex;
		justify-content: space-between;
		gap: 24px;
		padding: 30px;
		background:
			linear-gradient(135deg, rgba(255, 253, 249, 0.99) 0%, rgba(249, 243, 236, 0.96) 50%, rgba(240, 227, 209, 0.92) 100%),
			var(--color-panel);
		box-shadow: 0 20px 40px rgba(53, 39, 27, 0.08);
	}

	.overview-kicker,
	.composer-kicker,
	.list-kicker,
	.filter-panel-kicker {
		margin: 0 0 10px;
		font-size: 12px;
		letter-spacing: 0.16em;
		text-transform: uppercase;
		color: var(--color-muted);
	}

	.overview-card h1,
	.composer-head h2,
	.filter-head h2,
	.list-card-head h2,
	.modal-head h2 {
		margin: 0;
		font-family: var(--font-family-display);
		font-weight: 600;
	}

	.overview-card h1 {
		font-size: 42px;
	}

	.composer-head h2,
	.filter-head h2 {
		font-size: 30px;
	}

	.list-card-head h2,
	.modal-head h2 {
		font-size: 28px;
	}

	.overview-text,
	.composer-subtext,
	.filter-copy,
	.list-copy,
	.modal-copy {
		margin: 12px 0 0;
		color: var(--color-ink-soft);
		line-height: 1.8;
	}

	.overview-stats {
		display: grid;
		gap: 14px;
		min-width: 220px;
	}

	.overview-stats div {
		padding: 18px;
		border: 1px solid rgba(113, 91, 70, 0.13);
		border-radius: 18px;
		background: rgba(255, 250, 244, 0.9);
		box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.7);
	}

	.overview-stats span,
	.summary-item p,
	.composer-foot,
	.filter-search-field span,
	.filter-helper {
		display: block;
		margin: 0;
		font-size: 13px;
		color: var(--color-muted);
	}

	.overview-stats strong,
	.summary-item strong {
		display: block;
		margin-top: 8px;
		font-size: 24px;
		font-weight: 700;
	}

	.summary-grid {
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		gap: 16px;
	}

	.summary-item {
		padding: 20px;
		background: linear-gradient(180deg, rgba(255, 252, 247, 0.98) 0%, rgba(247, 241, 232, 0.9) 100%);
	}

	.composer-card,
	.filter-card {
		padding: 22px;
		background: rgba(255, 251, 246, 0.94);
		box-shadow: 0 16px 34px rgba(53, 39, 27, 0.07);
		display: grid;
		gap: 14px;
	}

	.composer-head,
	.filter-head,
	.list-card-head,
	.modal-head,
	.modal-actions {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		gap: 16px;
	}

	.composer-textarea {
		width: 100%;
		min-height: 152px;
		resize: vertical;
		padding: 16px 18px;
		border: 1px solid var(--color-border);
		border-radius: 20px;
		background: rgba(255, 255, 255, 0.86);
		outline: none;
		font-family: var(--font-family-base);
		line-height: 1.9;
		font-size: 1rem;
		color: var(--color-ink);
	}

	.composer-textarea:focus {
		border-color: rgba(140, 90, 60, 0.45);
		box-shadow: 0 0 0 4px rgba(140, 90, 60, 0.08);
	}

	.composer-foot {
		display: flex;
		justify-content: space-between;
		gap: 12px;
		flex-wrap: wrap;
	}

	.filter-mode-pill,
	.filter-sub-pill,
	.filter-panel-badge,
	.list-card-meta span,
	.diary-length,
	.delete-meta span {
		display: inline-flex;
		align-items: center;
		padding: 6px 10px;
		border-radius: 999px;
		background: rgba(140, 90, 60, 0.08);
		color: var(--color-muted);
		font-size: 13px;
	}

	.filter-mode-pill {
		align-self: flex-start;
		background: rgba(140, 90, 60, 0.12);
		color: var(--color-ink-soft);
	}

	.filter-sub-pill {
		background: rgba(113, 91, 70, 0.08);
	}

	.filter-status-group {
		display: flex;
		flex-wrap: wrap;
		justify-content: flex-end;
		gap: 8px;
	}

	.filter-grid {
		display: grid;
		grid-template-columns: minmax(0, 1.2fr) minmax(280px, 0.9fr) minmax(0, 1.1fr);
		gap: 16px;
	}

	.filter-panel {
		display: grid;
		gap: 14px;
		padding: 20px;
		border: 1px solid rgba(113, 91, 70, 0.11);
		border-radius: 22px;
		background:
			linear-gradient(180deg, rgba(255, 252, 247, 0.96) 0%, rgba(247, 241, 232, 0.86) 100%),
			rgba(255, 252, 247, 0.78);
		box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.72);
	}

	.filter-panel-compact {
		align-content: start;
		background:
			linear-gradient(180deg, rgba(251, 246, 239, 0.96) 0%, rgba(244, 235, 223, 0.9) 100%),
			rgba(255, 252, 247, 0.78);
	}

	.filter-panel-head {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		gap: 12px;
	}

	.filter-panel-head h3 {
		margin: 0;
		font-size: 1.1rem;
		font-weight: 600;
		color: var(--color-ink);
	}

	.filter-search-field {
		display: grid;
		gap: 8px;
	}

	.time-selector-card {
		display: grid;
		gap: 14px;
		padding: 16px;
		border-radius: 20px;
		background: rgba(247, 240, 231, 0.8);
		border: 1px solid rgba(113, 91, 70, 0.08);
	}

	.time-selector-section {
		display: grid;
		gap: 8px;
	}

	.time-selector-label {
		font-size: 13px;
		color: var(--color-muted);
	}

	.time-stepper {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr) auto;
		gap: 10px;
		align-items: stretch;
	}

	.time-stepper-button {
		white-space: nowrap;
	}

	.time-stepper-value {
		display: grid;
		place-items: center;
		padding: 12px;
		border-radius: 18px;
		background: rgba(255, 252, 247, 0.9);
		text-align: center;
	}

	.time-stepper-value strong {
		font-size: 1.05rem;
		color: var(--color-ink);
	}

	.time-stepper-value small {
		margin-top: 4px;
		font-size: 12px;
		color: var(--color-muted);
	}

	.sort-segment {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		padding: 4px;
		border-radius: 18px;
		border: 1px solid rgba(113, 91, 70, 0.14);
		background: rgba(255, 252, 247, 0.9);
		gap: 4px;
	}

	.sort-segment-button {
		min-height: 42px;
		padding: 0.7rem 0.9rem;
		border: none;
		border-radius: 14px;
		background: transparent;
		color: var(--color-muted);
		font-weight: 600;
		cursor: pointer;
		transition: background 0.2s ease, color 0.2s ease, transform 0.2s ease;
	}

	.sort-segment-button:hover:not(:disabled) {
		background: rgba(140, 90, 60, 0.08);
		color: var(--color-ink-soft);
	}

	.sort-segment-button:disabled {
		opacity: 0.55;
		cursor: not-allowed;
	}

	.sort-segment-active {
		background: var(--color-accent);
		color: white;
		box-shadow: 0 8px 18px rgba(140, 90, 60, 0.18);
	}

	.sort-tip {
		margin: 0;
		font-size: 13px;
		color: var(--color-muted);
		line-height: 1.7;
	}

	.filter-helper,
	.list-copy,
	.modal-copy {
		line-height: 1.7;
	}

	.filter-actions,
	.list-card-meta,
	.diary-row-foot,
	.diary-row-actions,
	.delete-meta,
	.pagination {
		display: flex;
		align-items: center;
		gap: 10px;
		flex-wrap: wrap;
	}

	.active-filters {
		display: flex;
		justify-content: space-between;
		gap: 12px;
		padding: 0 8px;
		font-size: 14px;
		color: var(--color-muted);
	}

	.list-card {
		overflow: hidden;
		background:
			linear-gradient(180deg, rgba(255, 253, 249, 0.98) 0%, rgba(247, 242, 235, 0.92) 100%),
			var(--color-panel);
	}

	.list-card-head {
		padding: 24px 28px 18px;
		border-bottom: 1px solid rgba(113, 91, 70, 0.08);
	}

	.diary-list {
		display: grid;
		gap: 14px;
		padding: 18px;
	}

	.diary-row {
		display: grid;
		grid-template-columns: 136px minmax(0, 1fr);
		gap: 24px;
		padding: 22px 24px;
		border: 1px solid rgba(113, 91, 70, 0.11);
		border-radius: 22px;
		background: rgba(255, 252, 247, 0.74);
		box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.72);
	}

	.diary-row-meta {
		display: grid;
		align-content: start;
		gap: 6px;
		padding: 12px 14px;
		border-radius: 18px;
		background: rgba(247, 240, 231, 0.92);
	}

	.diary-date,
	.diary-weekday,
	.diary-content,
	.delete-preview {
		margin: 0;
	}

	.diary-date {
		font-family: var(--font-family-mono);
		font-size: 14px;
		font-weight: 600;
		color: var(--color-ink);
		letter-spacing: 0.02em;
	}

	.diary-weekday {
		margin-top: 6px;
		font-size: 13px;
		color: var(--color-muted);
	}

	.diary-row-body {
		display: grid;
		gap: 16px;
	}

	.diary-content,
	.delete-preview {
		color: var(--color-ink-soft);
		line-height: 1.9;
		white-space: pre-wrap;
		word-break: break-word;
	}

	.diary-row-foot {
		justify-content: space-between;
	}

	.empty-state {
		padding: 72px 24px;
		text-align: center;
	}

	.empty-state p {
		margin: 0;
		color: var(--color-muted);
	}

	.pagination {
		justify-content: center;
		padding: 10px 24px 24px;
	}

	.month-pagination-shell {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 16px;
		padding: 0 24px 24px;
	}

	.month-pagination-copy {
		display: grid;
		gap: 6px;
	}

	.month-pagination-title,
	.month-pagination-subtitle {
		margin: 0;
	}

	.month-pagination-title {
		font-weight: 600;
		color: var(--color-ink);
	}

	.month-pagination-subtitle {
		font-size: 13px;
		color: var(--color-muted);
	}

	.month-pagination-actions {
		display: flex;
		gap: 10px;
		flex-wrap: wrap;
	}

	.pagination span {
		font-size: 14px;
		color: var(--color-muted);
	}

	.modal-backdrop {
		position: fixed;
		inset: 0;
		display: grid;
		place-items: center;
		padding: 24px;
		background: rgba(27, 22, 18, 0.42);
		z-index: 40;
	}

	.modal-card {
		width: min(100%, 520px);
		padding: 24px;
		display: grid;
		gap: 16px;
		background:
			linear-gradient(180deg, rgba(255, 252, 247, 0.98) 0%, rgba(246, 238, 227, 0.96) 100%),
			var(--color-panel);
	}

	.modal-head {
		align-items: center;
	}

	.delete-preview {
		padding: 16px;
		border-radius: 18px;
		background: rgba(247, 240, 231, 0.92);
	}

	.modal-card :global(.alert) {
		margin-bottom: 0;
	}

	@media (max-width: 960px) {
		.summary-grid,
		.filter-grid {
			grid-template-columns: 1fr 1fr;
		}

		.filter-panel-compact {
			grid-column: 1 / -1;
		}
	}

	@media (max-width: 900px) {
		.summary-grid,
		.filter-grid {
			grid-template-columns: 1fr;
		}

		.overview-card,
		.composer-head,
		.filter-head,
		.list-card-head,
		.diary-row {
			flex-direction: column;
		}

		.diary-row {
			grid-template-columns: 1fr;
		}
	}

	@media (max-width: 640px) {
		.diary-page {
			padding: 8px 0 28px;
			gap: 12px;
		}

		.overview-card,
		.summary-item,
		.composer-card,
		.filter-card,
		.list-card-head,
		.diary-row,
		.modal-card {
			padding-left: 16px;
			padding-right: 16px;
		}

		.overview-card {
			padding-top: 22px;
			padding-bottom: 22px;
		}

		.overview-card h1 {
			font-size: 34px;
		}

		.summary-grid {
			grid-template-columns: 1fr;
		}

		.composer-head,
		.filter-head,
		.active-filters,
		.list-card-meta,
		.composer-foot,
		.diary-row-foot,
		.modal-head,
		.modal-actions,
		.pagination {
			flex-direction: column;
			align-items: stretch;
		}

		.filter-actions :global(.btn),
		.diary-row-actions :global(.btn),
		.modal-actions :global(.btn) {
			width: 100%;
		}

		.filter-status-group {
			justify-content: flex-start;
		}

		.time-stepper {
			grid-template-columns: 1fr;
		}

		.sort-segment {
			grid-template-columns: 1fr;
		}

		.time-stepper-button,
		.sort-segment-button,
		.month-pagination-actions :global(.btn) {
			width: 100%;
		}

		.diary-list {
			padding: 12px;
		}

		.diary-row-meta,
		.filter-panel {
			padding: 12px;
		}

		.month-pagination-shell,
		.month-pagination-actions {
			flex-direction: column;
			align-items: stretch;
		}

		.modal-backdrop {
			padding: 12px;
		}
	}
</style>
