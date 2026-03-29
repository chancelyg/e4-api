<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { diaryAPI, type Diary, type DiaryStats } from '$lib/api';
	import { auth } from '$lib/stores.svelte';
	import { formatMonthLabel, formatWeekday, getCurrentMonthString, getMonthRange } from '$lib/date';

	let diaries = $state<Diary[]>([]);
	let stats = $state<DiaryStats | null>(null);
	let total = $state(0);
	let page = $state(1);
	let perPage = 50;
	let search = $state('');
	let month = $state('');
	let isLoading = $state(true);
	let monthPicker = $state(getCurrentMonthString());

	onMount(() => {
		void initPage();
	});

	async function initPage() {
		await auth.ensureInitialized();
		if (!auth.isLoggedIn) {
			goto('/');
			return;
		}
		loadData();
	}

	async function loadData() {
		isLoading = true;
		try {
			let startDate = '';
			let endDate = '';
			const normalizedSearch = search.trim();

			if (month) {
				const range = getMonthRange(month);
				startDate = range.startDate;
				endDate = range.endDate;
			}

			const [diariesResult, statsResult] = await Promise.all([
				diaryAPI.list({
					page,
					per_page: perPage,
					search: normalizedSearch,
					start_date: startDate || undefined,
					end_date: endDate || undefined
				}),
				diaryAPI.stats()
			]);
			search = normalizedSearch;
			diaries = diariesResult.diaries;
			total = diariesResult.total;
			stats = statsResult;
		} catch (err) {
			console.error('Failed to load data:', err);
			diaries = [];
			total = 0;
		} finally {
			isLoading = false;
		}
	}

	function handleSearch() {
		page = 1;
		loadData();
	}

	function clearFilters() {
		search = '';
		month = '';
		monthPicker = getCurrentMonthString();
		page = 1;
		loadData();
	}

	function applyMonthSelection(nextMonth: string) {
		month = nextMonth;
		monthPicker = nextMonth;
		page = 1;
		loadData();
	}

	function clearMonthSelection() {
		month = '';
		monthPicker = getCurrentMonthString();
		page = 1;
		loadData();
	}

	function getFrequencyPercent() {
		if (!stats || stats.total_count === 0 || stats.time_span_days === 0) return '0';
		return ((stats.total_count / stats.time_span_days) * 100).toFixed(1);
	}

	function getPageCount() {
		return Math.max(1, Math.ceil(total / perPage));
	}

	const pickerMonthLabel = $derived(formatMonthLabel(monthPicker));
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
				<p class="overview-text">按关键词、模糊片段与月份回看过去的记录，让信息先于装饰呈现。</p>
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

		<section class="filter-card card">
			<div class="filter-fields">
				<label class="filter-field filter-search">
					<span>搜索</span>
					<input
						type="text"
						bind:value={search}
						placeholder="支持整句、分词和模糊检索"
						onkeydown={(e) => e.key === 'Enter' && handleSearch()}
					/>
				</label>
				<label class="filter-field filter-month">
					<span>月份回看</span>
					<div class="month-filter-shell">
						<div class="month-filter-head month-filter-head-compact">
							<div>
								<strong>{pickerMonthLabel}</strong>
								<small>{month ? `当前筛选：${formatMonthLabel(month)}` : '选择年份和月份后点击检索'}</small>
							</div>
						</div>

						<div class="month-picker-row">
							<input
								class="month-input"
								type="month"
								bind:value={monthPicker}
								max={getCurrentMonthString()}
							/>
							<button type="button" class="btn variant-soft-surface" onclick={() => applyMonthSelection(monthPicker)}>
								按月份查询
							</button>
						</div>
					</div>
				</label>
			</div>
			<div class="filter-actions">
				<button type="button" onclick={handleSearch} class="btn variant-filled-primary">检索</button>
				{#if search || month}
					<button type="button" onclick={clearFilters} class="btn variant-soft-surface">清除全部</button>
				{/if}
				{#if month}
					<button type="button" onclick={clearMonthSelection} class="btn variant-soft-surface">仅清除月份</button>
				{/if}
				<a href="/diary/new" class="btn btn-compose">写新日记</a>
			</div>
		</section>

		{#if search || month}
			<div class="active-filters">
				{#if search && month}
					关键词“{search}” · {formatMonthLabel(month)}
				{:else if search}
					关键词“{search}”
				{:else}
					{formatMonthLabel(month)}
				{/if}
				<span>共 {total} 篇</span>
			</div>
		{/if}

		<section class="list-card card">
			{#if diaries.length === 0}
				<div class="empty-state">
					<p>{search || month ? '没有符合条件的日记。' : '还没有日记。'}</p>
					{#if !search && !month}
						<a href="/diary/new" class="btn variant-filled-primary">写第一篇日记</a>
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
								<a class="diary-edit-link" href={`/diary/${diary.id}`}>编辑</a>
							</div>
						</article>
					{/each}
				</div>

				{#if total > perPage}
					<div class="pagination">
						<button
							class="btn variant-soft-surface"
							disabled={page === 1}
							onclick={() => {
								page -= 1;
								loadData();
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
								loadData();
							}}
						>
							下一页
						</button>
					</div>
				{/if}
			{/if}
		</section>
	{/if}
</div>

<style>
	.diary-page {
		max-width: 1080px;
		margin: 0 auto;
		padding: 20px 0 40px;
	}

	.loading-container {
		display: flex;
		justify-content: center;
		padding: 72px 0;
	}

	.overview-card {
		display: flex;
		justify-content: space-between;
		gap: 24px;
		padding: 28px;
	}

	.overview-kicker {
		margin: 0 0 10px;
		font-size: 12px;
		letter-spacing: 0.16em;
		text-transform: uppercase;
		color: var(--color-muted);
	}

	.overview-card h1 {
		margin: 0;
		font-family: var(--font-family-display);
		font-size: 42px;
		font-weight: 600;
	}

	.overview-text {
		max-width: 38rem;
		margin: 12px 0 0;
		color: var(--color-ink-soft);
		line-height: 1.9;
	}

	.overview-stats {
		display: grid;
		gap: 14px;
		min-width: 220px;
	}

	.overview-stats div {
		padding: 16px 18px;
		border: 1px solid var(--color-border);
		border-radius: 18px;
		background: var(--color-panel-muted);
	}

	.overview-stats span,
	.summary-item p {
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
		gap: 14px;
		margin-top: 16px;
	}

	.summary-item {
		padding: 20px;
	}

	.filter-card {
		display: flex;
		justify-content: space-between;
		gap: 18px;
		padding: 20px;
		margin-top: 16px;
	}

	.filter-fields {
		display: grid;
		grid-template-columns: minmax(0, 1fr) minmax(340px, 420px);
		gap: 14px;
		flex: 1;
	}

	.filter-field {
		display: grid;
		gap: 8px;
	}

	.filter-field span {
		font-size: 13px;
		color: var(--color-muted);
	}

	.month-filter-shell {
		display: grid;
		gap: 14px;
		padding: 14px;
		border: 1px solid var(--color-border);
		border-radius: 18px;
		background: linear-gradient(180deg, rgba(255, 252, 247, 0.96) 0%, rgba(248, 241, 232, 0.92) 100%);
	}

	.month-filter-head {
		display: flex;
		justify-content: space-between;
		gap: 12px;
		align-items: flex-start;
	}

	.month-filter-head strong {
		display: block;
		font-size: 1rem;
		font-weight: 700;
	}

	.month-filter-head-compact {
		align-items: center;
	}

	.month-filter-head small {
		display: block;
		margin-top: 4px;
		color: var(--color-muted);
	}

	.month-picker-row {
		display: flex;
		align-items: center;
		gap: 10px;
		flex-wrap: wrap;
	}

	.month-input {
		min-height: 46px;
		padding: 0 14px;
		border: 1px solid var(--color-border);
		border-radius: 14px;
		background: rgba(255, 255, 255, 0.82);
		font: inherit;
		color: var(--color-ink);
	}

	.filter-field input {
		min-height: 46px;
		padding: 0 14px;
		border: 1px solid var(--color-border);
		border-radius: 14px;
		background: rgba(255, 255, 255, 0.82);
	}

	.filter-field input:focus {
		outline: none;
		border-color: rgba(140, 90, 60, 0.45);
		box-shadow: 0 0 0 4px rgba(140, 90, 60, 0.08);
	}

	.filter-actions {
		display: flex;
		align-items: end;
		gap: 10px;
		flex-wrap: wrap;
	}

	.btn-compose {
		background: var(--color-panel-muted);
		border-color: var(--color-border);
		color: var(--color-ink);
	}

	.active-filters {
		display: flex;
		justify-content: space-between;
		gap: 12px;
		margin-top: 12px;
		padding: 0 6px;
		font-size: 14px;
		color: var(--color-muted);
	}

	.list-card {
		margin-top: 16px;
		overflow: hidden;
	}

	.diary-list {
		display: grid;
	}

	.diary-row {
		display: grid;
		grid-template-columns: 160px minmax(0, 1fr);
		gap: 24px;
		padding: 22px 24px;
		border-top: 1px solid var(--color-border);
	}

	.diary-row:first-child {
		border-top: none;
	}

	.diary-row-meta {
		padding-top: 2px;
	}

	.diary-date,
	.diary-weekday,
	.diary-content {
		margin: 0;
	}

	.diary-date {
		font-family: var(--font-family-mono);
		font-size: 14px;
		font-weight: 600;
		color: var(--color-ink);
	}

	.diary-weekday {
		margin-top: 6px;
		font-size: 13px;
		color: var(--color-muted);
	}

	.diary-row-body {
		display: grid;
		gap: 12px;
	}

	.diary-content {
		color: var(--color-ink-soft);
		line-height: 1.9;
		white-space: pre-wrap;
		word-break: break-word;
	}

	.diary-edit-link {
		width: fit-content;
		padding-bottom: 2px;
		border-bottom: 1px solid var(--color-ink-soft);
		text-decoration: none;
		font-size: 14px;
		font-weight: 600;
		color: var(--color-ink);
	}

	.empty-state {
		padding: 72px 24px;
		text-align: center;
	}

	.empty-state p {
		margin: 0 0 16px;
		color: var(--color-muted);
	}

	.pagination {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 14px;
		padding: 20px 24px 24px;
		border-top: 1px solid var(--color-border);
	}

	.pagination span {
		font-size: 14px;
		color: var(--color-muted);
	}

	@media (max-width: 900px) {
		.summary-grid {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}

		.filter-card,
		.overview-card,
		.diary-row {
			grid-template-columns: 1fr;
			flex-direction: column;
		}

		.filter-fields {
			grid-template-columns: 1fr;
		}

		.filter-actions {
			align-items: stretch;
		}
	}

	@media (max-width: 640px) {
		.diary-page {
			padding: 8px 0 28px;
		}

		.overview-card,
		.summary-item,
		.filter-card,
		.diary-row {
			padding-left: 16px;
			padding-right: 16px;
		}

		.summary-grid {
			grid-template-columns: 1fr;
		}

		.active-filters,
		.pagination {
			flex-direction: column;
		}

		.filter-actions :global(.btn) {
			width: 100%;
		}
	}
</style>
