<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { diaryAPI, type Diary, type DiaryStats } from '$lib/api';
	import { auth } from '$lib/stores.svelte';
	import { formatMonthLabel, formatWeekday, getCurrentMonthString, getMonthRange, shiftMonth, shiftYear } from '$lib/date';

	let diaries = $state<Diary[]>([]);
	let stats = $state<DiaryStats | null>(null);
	let total = $state(0);
	let page = $state(1);
	const perPage = 50;
	let search = $state('');
	let month = $state('');
	let monthCursor = $state(getCurrentMonthString());
	let draftContent = $state('');
	let isLoading = $state(true);
	let isSaving = $state(false);
	let saveError = $state('');

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
		isLoading = true;
		try {
			const normalizedSearch = search.trim();
			const activeMonth = month || monthCursor;
			const range = month ? getMonthRange(activeMonth) : null;

			const [diariesResult, statsResult] = await Promise.all([
				diaryAPI.list({
					page,
					per_page: perPage,
					search: normalizedSearch,
					start_date: range?.startDate,
					end_date: range?.endDate
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

	async function handleCreateDiary() {
		const content = draftContent.trim();
		if (!content) {
			saveError = '请输入日记内容';
			return;
		}

		isSaving = true;
		saveError = '';

		try {
			await diaryAPI.create({ content });
			draftContent = '';
			page = 1;
			await loadData();
		} catch (err) {
			saveError = err instanceof Error ? err.message : '保存失败';
		} finally {
			isSaving = false;
		}
	}

	function handleSearch() {
		page = 1;
		month = monthCursor;
		void loadData();
	}

	function clearFilters() {
		search = '';
		month = '';
		monthCursor = getCurrentMonthString();
		page = 1;
		void loadData();
	}

	function shiftCursor(deltaMonths: number) {
		const nextMonth = shiftMonth(monthCursor, deltaMonths);
		monthCursor = nextMonth;
		page = 1;
		month = nextMonth;
		void loadData();
	}

	function shiftCursorYear(deltaYears: number) {
		const nextMonth = shiftYear(monthCursor, deltaYears);
		monthCursor = nextMonth;
		page = 1;
		month = nextMonth;
		void loadData();
	}

	function getFrequencyPercent() {
		if (!stats || stats.total_count === 0 || stats.time_span_days === 0) return '0';
		return ((stats.total_count / stats.time_span_days) * 100).toFixed(1);
	}

	function getPageCount() {
		return Math.max(1, Math.ceil(total / perPage));
	}

	const monthLabel = $derived(formatMonthLabel(monthCursor));
	const activeMonthLabel = $derived(month ? formatMonthLabel(month) : '全部月份');
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
				<p class="overview-text">写一小段信，马上入库。回看时按关键词和年月切换，不跳页，不打断，不截断内容。</p>
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
				onkeydown={(e) => {
					if ((e.metaKey || e.ctrlKey) && e.key === 'Enter' && !isSaving) {
						void handleCreateDiary();
					}
				}}
			></textarea>
			<div class="composer-foot">
				<span>当前 {draftContent.trim().length} / 300 字</span>
				<span>快捷键：Ctrl/Cmd + Enter</span>
			</div>
		</section>

		<section class="filter-card card">
			<div class="filter-bar">
				<label class="filter-search-field">
					<span>搜索</span>
					<input
						type="text"
						bind:value={search}
						placeholder="支持整句、分词和模糊检索"
						onkeydown={(e) => e.key === 'Enter' && handleSearch()}
					/>
				</label>

				<div class="month-strip" aria-label="月份回看">
					<span class="month-strip-label">月份回看</span>
					<button type="button" class="btn btn-sm variant-soft-surface" onclick={() => shiftCursorYear(-1)}>«</button>
					<button type="button" class="btn btn-sm variant-soft-surface" onclick={() => shiftCursor(-1)}>‹</button>
					<div class="month-chip">
						<strong>{monthLabel}</strong>
						<small>{activeMonthLabel}</small>
					</div>
					<button type="button" class="btn btn-sm variant-soft-surface" onclick={() => shiftCursor(1)}>›</button>
					<button type="button" class="btn btn-sm variant-soft-surface" onclick={() => shiftCursorYear(1)}>»</button>
				</div>

				<div class="filter-actions">
					<button type="button" onclick={handleSearch} class="btn variant-filled-primary">查询</button>
					{#if search || month}
						<button type="button" onclick={clearFilters} class="btn variant-soft-surface">清除</button>
					{/if}
				</div>
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
			<div class="list-card-head">
				<div>
					<p class="list-kicker">Entries</p>
					<h2>{search || month ? '筛选结果' : '最近记录'}</h2>
				</div>
				<div class="list-card-meta">
					<span>共 {total} 篇</span>
					<span>本页 {diaries.length} 篇</span>
				</div>
			</div>

			{#if diaries.length === 0}
				<div class="empty-state">
					<p>{search || month ? '没有符合条件的日记。' : '还没有日记。先写下第一条吧。'}</p>
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
								</div>
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
	.list-kicker {
		margin: 0 0 10px;
		font-size: 12px;
		letter-spacing: 0.16em;
		text-transform: uppercase;
		color: var(--color-muted);
	}

	.overview-card h1,
	.composer-head h2,
	.list-card-head h2 {
		margin: 0;
		font-family: var(--font-family-display);
		font-weight: 600;
	}

	.overview-card h1 {
		font-size: 42px;
	}

	.composer-head h2 {
		font-size: 30px;
	}

	.list-card-head h2 {
		font-size: 28px;
	}

	.overview-text,
	.composer-subtext {
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
		padding: 18px 18px;
		border: 1px solid rgba(113, 91, 70, 0.13);
		border-radius: 18px;
		background: rgba(255, 250, 244, 0.9);
		box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.7);
	}

	.overview-stats span,
	.summary-item p,
	.composer-foot,
	.month-strip-label,
	.month-chip small {
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
	}

	.composer-card {
		display: grid;
		gap: 14px;
	}

	.composer-head {
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

	.composer-textarea:focus,
	.filter-search-field input:focus {
		border-color: rgba(140, 90, 60, 0.45);
		box-shadow: 0 0 0 4px rgba(140, 90, 60, 0.08);
	}

	.composer-foot {
		display: flex;
		justify-content: space-between;
		gap: 12px;
		flex-wrap: wrap;
	}

	.filter-bar {
		display: grid;
		grid-template-columns: minmax(0, 1.2fr) auto auto;
		gap: 16px;
		align-items: end;
	}

	.filter-search-field {
		display: grid;
		gap: 8px;
	}

	.filter-search-field span {
		font-size: 13px;
		color: var(--color-muted);
	}

	.filter-search-field input {
		min-height: 46px;
		padding: 0 14px;
		border: 1px solid var(--color-border);
		border-radius: 14px;
		background: rgba(255, 255, 255, 0.82);
	}

	.month-strip {
		display: grid;
		grid-auto-flow: column;
		grid-auto-columns: max-content;
		gap: 10px;
		align-items: center;
		padding: 10px 12px;
		border: 1px solid var(--color-border);
		border-radius: 18px;
		background: linear-gradient(180deg, rgba(255, 252, 247, 0.96) 0%, rgba(248, 241, 232, 0.92) 100%);
	}

	.month-strip-label {
		margin-right: 4px;
	}

	.month-chip {
		min-width: 124px;
		padding: 0 4px;
		text-align: center;
	}

	.month-chip strong {
		display: block;
		font-size: 1rem;
		font-weight: 700;
	}

	.filter-actions {
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

	.list-card-head {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		gap: 16px;
		padding: 24px 28px 18px;
		border-bottom: 1px solid rgba(113, 91, 70, 0.08);
	}

	.list-card-meta {
		display: flex;
		gap: 10px;
		flex-wrap: wrap;
		justify-content: flex-end;
	}

	.list-card-meta span,
	.diary-length {
		display: inline-flex;
		align-items: center;
		padding: 6px 10px;
		border-radius: 999px;
		background: rgba(140, 90, 60, 0.08);
		color: var(--color-muted);
		font-size: 13px;
	}

	.list-card {
		overflow: hidden;
		background:
			linear-gradient(180deg, rgba(255, 253, 249, 0.98) 0%, rgba(247, 242, 235, 0.92) 100%),
			var(--color-panel);
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
	.diary-content {
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

	.diary-content {
		color: var(--color-ink-soft);
		line-height: 1.9;
		white-space: pre-wrap;
		word-break: break-word;
	}

	.diary-row-foot {
		display: flex;
		align-items: center;
		justify-content: flex-start;
		gap: 12px;
		flex-wrap: wrap;
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
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 14px;
		padding: 10px 24px 24px;
	}

	.pagination span {
		font-size: 14px;
		color: var(--color-muted);
	}

	@media (max-width: 960px) {
		.summary-grid {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}

		.filter-bar {
			grid-template-columns: 1fr;
		}

		.month-strip {
			justify-content: start;
			overflow-x: auto;
		}
	}

	@media (max-width: 900px) {
		.overview-card,
		.composer-head,
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
		.diary-row {
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
		.active-filters,
		.list-card-meta,
		.composer-foot,
		.pagination {
			flex-direction: column;
			align-items: stretch;
		}

		.filter-actions :global(.btn) {
			width: 100%;
		}

		.diary-list {
			padding: 12px;
		}

		.diary-row-meta {
			padding: 10px 12px;
		}
	}
</style>
