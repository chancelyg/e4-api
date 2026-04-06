<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { adminJSONAPI, type JSONStoreItemMeta } from '$lib/api';
	import { auth } from '$lib/stores.svelte';

	type DeleteState = {
		item: JSONStoreItemMeta | null;
		keyInput: string;
		error: string;
	};

	let items = $state<JSONStoreItemMeta[]>([]);
	let total = $state(0);
	let totalSizeBytes = $state(0);
	let newestKey = $state('暂无');
	let page = $state(1);
	const perPage = 20;
	let search = $state('');
	let appliedSearch = $state('');
	let sortOrder = $state<'asc' | 'desc'>('desc');
	let isLoading = $state(true);
	let isRefreshing = $state(false);
	let isCopyingKey = $state('');
	let copiedKey = $state('');
	let error = $state('');
	let showDeleteModal = $state(false);
	let deleteState = $state<DeleteState>({ item: null, keyInput: '', error: '' });
	let isDeleting = $state(false);

	onMount(() => {
		void initPage();
	});

	$effect(() => {
		if (!showDeleteModal) {
			return;
		}

		const handleKeydown = (event: KeyboardEvent) => {
			if (event.key === 'Escape') {
				closeDeleteModal();
			}
		};

		window.addEventListener('keydown', handleKeydown);
		return () => window.removeEventListener('keydown', handleKeydown);
	});

	async function initPage() {
		await auth.ensureInitialized();
		if (!auth.isLoggedIn) {
			goto('/');
			return;
		}
		await loadItems();
	}

	async function loadItems(options?: { silent?: boolean }) {
		const silent = options?.silent ?? false;
		if (silent && items.length > 0) {
			isRefreshing = true;
		} else {
			isLoading = true;
		}
		error = '';

		try {
			const result = await adminJSONAPI.list({
				page,
				per_page: perPage,
				search: appliedSearch,
				sort: sortOrder
			});
			items = result.items;
			total = result.total;
			totalSizeBytes = result.total_size_bytes;
			newestKey = result.newest_key || '暂无';
			sortOrder = result.sort_order;
		} catch (err) {
			error = err instanceof Error ? err.message : '加载 JSON 列表失败';
			if (!silent) {
				items = [];
				total = 0;
				totalSizeBytes = 0;
				newestKey = '暂无';
			}
		} finally {
			isLoading = false;
			isRefreshing = false;
		}
	}

	async function copyJSON(item: JSONStoreItemMeta) {
		isCopyingKey = item.key;
		copiedKey = '';
		try {
			const result = await adminJSONAPI.getContent(item.key);
			await navigator.clipboard.writeText(result.content);
			copiedKey = item.key;
			window.setTimeout(() => {
				if (copiedKey === item.key) {
					copiedKey = '';
				}
			}, 1800);
		} catch (err) {
			error = err instanceof Error ? err.message : '复制 JSON 失败';
		} finally {
			isCopyingKey = '';
		}
	}

	function handleSearch() {
		appliedSearch = search.trim();
		page = 1;
		void loadItems();
	}

	function clearSearch() {
		search = '';
		appliedSearch = '';
		page = 1;
		void loadItems();
	}

	function toggleSortOrder() {
		sortOrder = sortOrder === 'desc' ? 'asc' : 'desc';
		page = 1;
		void loadItems();
	}

	function openDeleteModal(item: JSONStoreItemMeta) {
		deleteState = { item, keyInput: '', error: '' };
		showDeleteModal = true;
	}

	function closeDeleteModal() {
		showDeleteModal = false;
		deleteState = { item: null, keyInput: '', error: '' };
	}

	async function confirmDelete() {
		if (!deleteState.item) {
			return;
		}
		if (deleteState.keyInput.trim() !== deleteState.item.key) {
			deleteState = { ...deleteState, error: '请输入完整 key 后再删除' };
			return;
		}

		isDeleting = true;
		try {
			await adminJSONAPI.delete(deleteState.item.key);
			closeDeleteModal();
			if (items.length === 1 && page > 1) {
				page -= 1;
			}
			await loadItems({ silent: true });
		} catch (err) {
			deleteState = {
				...deleteState,
				error: err instanceof Error ? err.message : '删除 JSON 失败'
			};
		} finally {
			isDeleting = false;
		}
	}

	function getPageCount() {
		return Math.max(1, Math.ceil(total / perPage));
	}

	function formatDateTime(value: string) {
		return new Intl.DateTimeFormat('zh-CN', {
			year: 'numeric',
			month: '2-digit',
			day: '2-digit',
			hour: '2-digit',
			minute: '2-digit'
		}).format(new Date(value));
	}

	function formatSize(sizeBytes: number) {
		if (sizeBytes < 1024) {
			return `${sizeBytes} B`;
		}
		if (sizeBytes < 1024 * 1024) {
			return `${(sizeBytes / 1024).toFixed(1)} KiB`;
		}
		return `${(sizeBytes / (1024 * 1024)).toFixed(2)} MiB`;
	}

	function getRemainingDays(item: JSONStoreItemMeta) {
		if (item.is_expired) {
			return '已过期';
		}
		const diff = new Date(item.expires_at).getTime() - Date.now();
		const days = Math.max(1, Math.ceil(diff / (24 * 60 * 60 * 1000)));
		return `${days} 天`;
	}

	const activeCount = $derived(items.filter((item) => !item.is_expired).length);
	const expiredCount = $derived(items.filter((item) => item.is_expired).length);
	const sortButtonText = $derived(sortOrder === 'desc' ? '由近到远' : '由远到近');
</script>

<div class="json-page">
	{#if isLoading}
		<div class="loading-state card">
			<p class="text-surface-500">加载中...</p>
		</div>
	{:else}
		<div class:page-refreshing={isRefreshing} class="json-content">
			{#if isRefreshing}
				<div class="refresh-indicator">正在刷新列表...</div>
			{/if}

			{#if error}
				<div class="alert variant-soft-error">
					<p>{error}</p>
				</div>
			{/if}

			<section class="json-hero card">
				<div>
					<p class="section-kicker">JSON Shelf</p>
					<h1>JSON</h1>
					<p class="section-copy">这里管理一组匿名开放的 JSON 临时存取接口。外部调用方只需要准备 key 和 JSON 正文，就可以用 GET、POST、PUT、DELETE 快速读写。</p>
				</div>
				<div class="hero-stats">
					<div class="hero-stat">
						<span>当前页总量</span>
						<strong>{items.length}</strong>
					</div>
					<div class="hero-stat">
						<span>未过期</span>
						<strong>{activeCount}</strong>
					</div>
					<div class="hero-stat">
						<span>已过期</span>
						<strong>{expiredCount}</strong>
					</div>
				</div>
			</section>

			<section class="summary-grid">
				<article class="summary-item card">
					<p>总条数</p>
					<strong>{total}</strong>
				</article>
				<article class="summary-item card">
					<p>总大小</p>
					<strong>{formatSize(totalSizeBytes)}</strong>
				</article>
				<article class="summary-item card">
					<p>最新入库</p>
					<strong>{newestKey}</strong>
				</article>
				<article class="summary-item card">
					<p>排序方式</p>
					<button type="button" class="btn variant-soft-surface summary-sort-button" onclick={toggleSortOrder}>{sortButtonText}</button>
				</article>
			</section>

			<section class="filter-card card">
				<div class="filter-bar">
					<label class="filter-search-field">
						<span>搜索</span>
						<input
							type="text"
							bind:value={search}
							placeholder="支持模糊搜索 ID、key 和 JSON 内容"
							onkeydown={(event) => event.key === 'Enter' && handleSearch()}
						/>
					</label>

					<div class="filter-actions">
						<button type="button" onclick={handleSearch} class="btn variant-filled-primary">查询</button>
						{#if appliedSearch}
							<button type="button" onclick={clearSearch} class="btn variant-soft-surface">清除</button>
						{/if}
					</div>
				</div>
			</section>

			{#if appliedSearch}
				<div class="active-filters">
					<span>关键词“{appliedSearch}”</span>
					<span>共 {total} 条</span>
				</div>
			{/if}

			<section class="list-card card">
				<div class="list-head">
					<div>
						<p class="section-kicker">Entries</p>
						<h2>{appliedSearch ? '筛选结果' : '存储记录'}</h2>
						<p class="section-copy">复制会单独读取原始 JSON，页面上只保留检索和生命周期所需的信息。删除需要再次输入完整 key。</p>
					</div>
					<div class="list-meta">
						<span>共 {total} 条</span>
						<span>本页 {items.length} 条</span>
					</div>
				</div>

				{#if items.length === 0}
					<div class="empty-state">
						<p>{appliedSearch ? '没有匹配的 JSON 记录。' : '还没有 JSON 记录。'}</p>
					</div>
				{:else}
					<div class="json-list">
						{#each items as item (item.key)}
							<article class:json-card-expired={item.is_expired} class="json-card card">
								<div class="json-card-head">
									<div>
										<p class="section-kicker">JSON Item</p>
										<h3>{item.key}</h3>
										<p class="json-card-id">ID #{item.id}</p>
									</div>
									<div class="json-card-actions">
										<button class="btn variant-soft-surface" type="button" onclick={() => copyJSON(item)} disabled={isCopyingKey === item.key}>
											{#if isCopyingKey === item.key}
												复制中...
											{:else if copiedKey === item.key}
												已复制
											{:else}
												复制 JSON
											{/if}
										</button>
										<button class="btn btn-quiet" type="button" onclick={() => openDeleteModal(item)}>删除</button>
									</div>
								</div>

								<div class="json-card-meta-grid">
									<div class="meta-item">
										<small>大小</small>
										<strong>{formatSize(item.size_bytes)}</strong>
									</div>
									<div class="meta-item">
										<small>创建时间</small>
										<strong>{formatDateTime(item.created_at)}</strong>
									</div>
									<div class="meta-item">
										<small>更新时间</small>
										<strong>{formatDateTime(item.updated_at)}</strong>
									</div>
									<div class="meta-item">
										<small>剩余有效时间</small>
										<strong>{getRemainingDays(item)}</strong>
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
									void loadItems();
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
									void loadItems();
								}}
							>
								下一页
							</button>
						</div>
					{/if}
				{/if}
			</section>
		</div>
	{/if}

	{#if showDeleteModal && deleteState.item}
		<div class="modal-backdrop" role="presentation" onclick={(event) => event.target === event.currentTarget && closeDeleteModal()}>
			<div class="modal-card card delete-modal" role="dialog" aria-modal="true" aria-labelledby="json-delete-title">
				<div class="modal-head">
					<div>
						<p class="section-kicker">Delete JSON</p>
						<h2 id="json-delete-title">删除 JSON</h2>
					</div>
					<button class="btn btn-quiet" type="button" onclick={closeDeleteModal}>关闭</button>
				</div>
				<p class="section-copy">这会直接删除存储记录。为了防止误删，请再次输入完整 key。</p>
				<p class="delete-key-name">请输入 <strong>{deleteState.item.key}</strong> 确认删除。</p>
				<input class="input" type="text" bind:value={deleteState.keyInput} placeholder="输入完整 key" />
				{#if deleteState.error}
					<div class="alert variant-soft-error">
						<p>{deleteState.error}</p>
					</div>
				{/if}
				<div class="modal-actions">
					<button type="button" class="btn variant-soft-surface" onclick={closeDeleteModal}>取消</button>
					<button type="button" class="btn variant-filled-primary" onclick={confirmDelete} disabled={isDeleting}>{isDeleting ? '删除中...' : '确认删除'}</button>
				</div>
			</div>
		</div>
	{/if}
</div>

<style>
	.json-page {
		max-width: 1120px;
		margin: 0 auto;
		padding: 12px 0 40px;
		display: grid;
		gap: 20px;
	}

	.json-content {
		display: grid;
		gap: 20px;
	}

	.page-refreshing {
		opacity: 0.92;
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

	.loading-state,
	.empty-state {
		padding: 56px 24px;
		text-align: center;
	}

	.json-hero,
	.filter-card,
	.list-card,
	.modal-card,
	.summary-item,
	.json-card {
		background:
			linear-gradient(180deg, rgba(255, 252, 247, 0.98) 0%, rgba(246, 238, 227, 0.96) 100%),
			var(--color-panel);
	}

	.json-hero {
		display: flex;
		justify-content: space-between;
		gap: 24px;
		padding: 30px;
		background:
			linear-gradient(135deg, rgba(255, 253, 249, 0.99) 0%, rgba(249, 243, 236, 0.96) 48%, rgba(239, 227, 210, 0.92) 100%),
			var(--color-panel);
		box-shadow: 0 20px 42px rgba(83, 58, 35, 0.08);
	}

	.section-kicker {
		margin: 0 0 8px;
		font-size: 12px;
		letter-spacing: 0.16em;
		text-transform: uppercase;
		color: var(--color-muted);
	}

	.json-hero h1,
	.list-head h2,
	.modal-head h2,
	.json-card-head h3 {
		margin: 0;
		font-family: var(--font-family-display);
		font-weight: 600;
	}

	.json-hero h1 {
		font-size: 42px;
		letter-spacing: -0.03em;
	}

	.list-head h2 {
		font-size: 28px;
	}

	.section-copy,
	.json-card-id {
		margin: 12px 0 0;
		line-height: 1.72;
		color: var(--color-ink-soft);
	}

	.hero-stats {
		display: grid;
		gap: 14px;
		min-width: 220px;
	}

	.hero-stat,
	.summary-item {
		padding: 18px;
		border: 1px solid rgba(113, 91, 70, 0.13);
		border-radius: 18px;
		background: rgba(255, 250, 244, 0.9);
		box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.7);
	}

	.hero-stat span,
	.summary-item p,
	.list-meta span,
	.meta-item small,
	.filter-search-field span {
		display: block;
		margin: 0;
		font-size: 13px;
		color: var(--color-muted);
	}

	.hero-stat strong,
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

	.summary-sort-button {
		margin-top: 12px;
		width: 100%;
	}

	.filter-card,
	.list-card {
		box-shadow: 0 16px 34px rgba(53, 39, 27, 0.07);
	}

	.filter-card {
		padding: 22px;
	}

	.filter-bar,
	.list-head,
	.modal-head,
	.modal-actions,
	.json-card-head,
	.json-card-actions {
		display: flex;
		justify-content: space-between;
		gap: 16px;
		align-items: flex-start;
	}

	.filter-bar {
		align-items: end;
	}

	.filter-search-field {
		flex: 1;
		display: grid;
		gap: 8px;
	}

	.filter-search-field input {
		min-height: 46px;
		padding: 0 14px;
		border: 1px solid var(--color-border);
		border-radius: 14px;
		background: rgba(255, 255, 255, 0.82);
	}

	.filter-search-field input:focus {
		outline: none;
		border-color: rgba(140, 90, 60, 0.45);
		box-shadow: 0 0 0 4px rgba(140, 90, 60, 0.08);
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

	.list-head {
		padding: 24px 28px 18px;
		border-bottom: 1px solid rgba(113, 91, 70, 0.08);
	}

	.list-meta {
		display: flex;
		gap: 10px;
		flex-wrap: wrap;
		justify-content: flex-end;
	}

	.list-meta span {
		display: inline-flex;
		align-items: center;
		padding: 6px 10px;
		border-radius: 999px;
		background: rgba(140, 90, 60, 0.08);
	}

	.json-list {
		display: grid;
		gap: 14px;
		padding: 18px;
	}

	.json-card {
		padding: 22px;
		border-color: rgba(113, 91, 70, 0.12);
		box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.72);
	}

	.json-card-expired {
		opacity: 0.86;
	}

	.json-card-id {
		font-family: var(--font-family-mono);
		font-size: 0.9rem;
	}

	.json-card-actions {
		align-items: center;
		flex-wrap: wrap;
	}

	.json-card-meta-grid {
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		gap: 14px;
		margin-top: 18px;
	}

	.meta-item {
		padding: 16px;
		border: 1px solid rgba(113, 91, 70, 0.1);
		border-radius: 18px;
		background: rgba(255, 252, 247, 0.74);
	}

	.meta-item strong {
		display: block;
		margin-top: 8px;
		font-size: 0.96rem;
		color: var(--color-ink-soft);
		line-height: 1.5;
		word-break: break-word;
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

	.modal-backdrop {
		position: fixed;
		inset: 0;
		z-index: 50;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 20px;
		background: rgba(27, 22, 18, 0.42);
		backdrop-filter: blur(10px);
	}

	.modal-card {
		width: min(560px, 100%);
		padding: 24px;
		border-radius: 24px;
		box-shadow: 0 22px 48px rgba(27, 22, 18, 0.2);
	}

	.delete-key-name {
		margin: 0 0 16px;
		color: var(--color-ink-soft);
	}

	@media (max-width: 1080px) {
		.summary-grid,
		.json-card-meta-grid {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
	}

	@media (max-width: 900px) {
		.json-hero,
		.filter-bar,
		.list-head,
		.modal-head,
		.modal-actions,
		.json-card-head {
			flex-direction: column;
		}
	}

	@media (max-width: 640px) {
		.json-page {
			padding: 8px 0 28px;
			gap: 12px;
		}

		.json-hero,
		.filter-card,
		.summary-item,
		.list-head,
		.json-card,
		.modal-card {
			padding-left: 16px;
			padding-right: 16px;
		}

		.json-hero h1 {
			font-size: 34px;
		}

		.summary-grid,
		.json-card-meta-grid {
			grid-template-columns: 1fr;
		}

		.filter-actions,
		.active-filters,
		.list-meta,
		.json-card-actions,
		.pagination {
			flex-direction: column;
			align-items: stretch;
		}

		.filter-actions :global(.btn),
		.json-card-actions :global(.btn) {
			width: 100%;
		}

		.json-list {
			padding: 12px;
		}
	}
</style>
