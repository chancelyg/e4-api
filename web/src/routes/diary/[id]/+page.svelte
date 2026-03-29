<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { diaryAPI, type Diary } from '$lib/api';
	import { formatLongDate } from '$lib/date';

	let diary = $state<Diary | null>(null);
	let isEditing = $state(false);
	let editContent = $state('');
	let editDate = $state('');
	let isLoading = $state(true);
	let isSaving = $state(false);
	let showDeleteConfirm = $state(false);
	let error = $state('');

	onMount(async () => {
		const idParam = page.params.id;
		if (!idParam) {
			goto('/diary');
			return;
		}

		const id = parseInt(idParam, 10);
		if (isNaN(id)) {
			goto('/diary');
			return;
		}

		try {
			diary = await diaryAPI.get(id);
			editContent = diary.content;
			editDate = diary.create_date;
		} catch {
			goto('/diary');
		} finally {
			isLoading = false;
		}
	});

	async function handleSave() {
		if (!diary) return;
		if (!editContent.trim()) {
			error = '请输入日记内容';
			return;
		}

		isSaving = true;
		error = '';
		try {
			await diaryAPI.update(diary.id, {
				content: editContent.trim(),
				create_date: editDate
			});
			diary.content = editContent.trim();
			diary.create_date = editDate;
			isEditing = false;
		} catch (err) {
			error = err instanceof Error ? err.message : '保存失败';
		} finally {
			isSaving = false;
		}
	}

	async function handleDelete() {
		if (!diary) return;

		try {
			await diaryAPI.delete(diary.id);
			goto('/diary');
		} catch (err) {
			error = err instanceof Error ? err.message : '删除失败';
		}
	}
</script>

<div class="detail-page">
	{#if isLoading}
		<div class="text-center py-12">
			<p class="text-surface-500">加载中...</p>
		</div>
	{:else if diary}
		<div class="detail-head">
			<div>
				<p class="detail-kicker">Entry</p>
				<h2>{formatLongDate(diary.create_date)}</h2>
			</div>
			<div class="detail-actions">
				{#if isEditing}
					<button onclick={() => (isEditing = false)} class="btn variant-soft-surface">取消</button>
					<button onclick={handleSave} class="btn variant-filled-primary" disabled={isSaving}>
						{isSaving ? '保存中...' : '保存'}
					</button>
				{:else}
					<button onclick={() => (showDeleteConfirm = true)} class="btn variant-soft-error">删除</button>
					<button onclick={() => (isEditing = true)} class="btn variant-filled-primary">编辑</button>
				{/if}
			</div>
		</div>

		{#if error}
			<div class="alert variant-soft-error">
				<p>{error}</p>
			</div>
		{/if}

		{#if isEditing}
			<section class="card detail-card">
				<label class="detail-date-field">
					<span>日期</span>
					<input type="date" bind:value={editDate} class="input detail-date-input" />
				</label>
				<textarea bind:value={editContent} class="editor-textarea"></textarea>
			</section>
		{:else}
			<section class="card detail-card detail-content-card">
				<p class="detail-content">{diary.content}</p>
			</section>
		{/if}

		{#if showDeleteConfirm}
			<div class="fixed inset-0 bg-overlay flex items-center justify-center z-50">
				<div class="card confirm-card">
					<h3>确认删除</h3>
					<p>确定要删除这篇日记吗？此操作无法撤销。</p>
					<div class="confirm-actions">
						<button onclick={() => (showDeleteConfirm = false)} class="btn variant-soft-surface">取消</button>
						<button onclick={handleDelete} class="btn variant-filled-error">确认删除</button>
					</div>
				</div>
			</div>
		{/if}
	{/if}
</div>

<style>
	.detail-page {
		max-width: 920px;
		margin: 0 auto;
		padding: 20px 0 40px;
	}

	.detail-head {
		display: flex;
		justify-content: space-between;
		align-items: end;
		gap: 16px;
		margin-bottom: 16px;
	}

	.detail-kicker {
		margin: 0 0 8px;
		font-size: 12px;
		letter-spacing: 0.16em;
		text-transform: uppercase;
		color: var(--color-muted);
	}

	.detail-head h2 {
		margin: 0;
		font-family: var(--font-family-display);
		font-size: 2.2rem;
		font-weight: 600;
	}

	.detail-actions,
	.confirm-actions {
		display: flex;
		gap: 10px;
		flex-wrap: wrap;
	}

	.detail-card {
		padding: 22px;
	}

	.detail-content-card {
		min-height: 52vh;
	}

	.detail-date-field {
		display: inline-grid;
		gap: 8px;
		margin-bottom: 18px;
	}

	.detail-date-field span {
		font-size: 13px;
		color: var(--color-muted);
	}

	.detail-date-input {
		width: 220px;
	}

	.detail-content {
		margin: 0;
		font-size: 1.02rem;
		line-height: 2;
		white-space: pre-wrap;
		color: var(--color-ink);
	}

	.confirm-card {
		width: min(92vw, 420px);
		padding: 22px;
	}

	.confirm-card h3,
	.confirm-card p {
		margin: 0;
	}

	.confirm-card p {
		margin-top: 10px;
		color: var(--color-muted);
		line-height: 1.8;
	}

	.confirm-actions {
		justify-content: end;
		margin-top: 20px;
	}

	@media (max-width: 640px) {
		.detail-page {
			padding: 8px 0 28px;
		}

		.detail-head {
			flex-direction: column;
			align-items: stretch;
		}

		.detail-actions :global(.btn),
		.confirm-actions :global(.btn),
		.detail-date-input {
			width: 100%;
		}

		.detail-card,
		.confirm-card {
			padding: 16px;
		}
	}
</style>
