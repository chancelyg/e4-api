<script lang="ts">
	import '../app.css';
	import { page } from '$app/state';
	import { auth } from '$lib/stores.svelte';
	import { onMount } from 'svelte';

	let { children } = $props();

	onMount(() => {
		auth.ensureInitialized();
	});

	async function handleLogout() {
		await auth.logout();
		window.location.href = '/';
	}

	const isDiarySection = $derived(page.url.pathname.startsWith('/diary'));
	const isGoalsSection = $derived(page.url.pathname.startsWith('/goals'));
</script>

<div class="app-shell">
	{#if auth.isLoggedIn}
		<aside class="app-sidebar">
			<a class="sidebar-brand" href="/diary">
				<p class="sidebar-kicker">E4 Diary</p>
				<h1>记录与回看</h1>
				<p class="sidebar-meta">把日常整理成清晰、可检索的个人档案。</p>
			</a>

			<nav class="sidebar-nav">
				<a href="/diary" class:sidebar-link-active={isDiarySection} class="sidebar-link">日记</a>
				<a href="/goals" class:sidebar-link-active={isGoalsSection} class="sidebar-link sidebar-link-secondary">目标</a>
			</nav>

			<div class="sidebar-footer">
				<div>
					<p class="sidebar-user-label">当前用户</p>
					<p class="sidebar-user-name">{auth.username}</p>
				</div>
				<button onclick={handleLogout} class="btn btn-quiet">退出</button>
			</div>
		</aside>
	{/if}

	<main class="app-content" class:app-content-wide={!auth.isLoggedIn}>
		{#if auth.isLoading && !auth.hasInitialized}
			<div class="page-loading">正在初始化...</div>
		{:else}
			{@render children()}
		{/if}
	</main>
</div>

<style>
	.app-shell {
		display: grid;
		grid-template-columns: 15rem minmax(0, 1fr);
		min-height: 100vh;
		max-width: 1440px;
		margin: 0 auto;
		padding: 20px;
		gap: 20px;
	}

	.app-sidebar {
		display: flex;
		flex-direction: column;
		justify-content: space-between;
		gap: 20px;
		padding: 20px;
		border: 1px solid rgba(113, 91, 70, 0.16);
		border-radius: 28px;
		background:
			linear-gradient(180deg, rgba(255, 253, 249, 0.98) 0%, rgba(247, 241, 232, 0.94) 68%, rgba(243, 233, 220, 0.92) 100%),
			var(--color-panel);
		box-shadow: var(--shadow-soft);
		position: sticky;
		top: 20px;
		height: calc(100vh - 40px);
		backdrop-filter: blur(10px);
	}

	.sidebar-brand {
		display: block;
		text-decoration: none;
		color: inherit;
	}

	.sidebar-kicker {
		margin: 0 0 10px;
		font-size: 12px;
		letter-spacing: 0.16em;
		text-transform: uppercase;
		color: var(--color-muted);
	}

	.sidebar-brand h1 {
		margin: 0;
		font-family: var(--font-family-display);
		font-size: 28px;
		font-weight: 600;
		line-height: 1.1;
	}

	.sidebar-meta {
		margin: 12px 0 0;
		font-size: 14px;
		line-height: 1.7;
		color: var(--color-muted);
	}

	.sidebar-nav {
		display: grid;
		gap: 10px;
	}

	.sidebar-link {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 13px 15px;
		border: 1px solid rgba(113, 91, 70, 0.14);
		border-radius: 16px;
		background: rgba(255, 251, 246, 0.86);
		text-decoration: none;
		font-size: 15px;
		font-weight: 600;
		transition: border-color 0.2s ease, background 0.2s ease, transform 0.2s ease;
		box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.55);
	}

	.sidebar-link::after {
		content: '->';
		font-size: 13px;
		color: var(--color-muted);
	}

	.sidebar-link:hover {
		transform: translateY(-1px);
		border-color: var(--color-ink-soft);
		background: #f4f0e9;
	}

	.sidebar-link-active {
		border-color: rgba(140, 90, 60, 0.3);
		background: linear-gradient(180deg, rgba(240, 224, 206, 0.96) 0%, rgba(233, 217, 199, 0.92) 100%);
		box-shadow: inset 0 0 0 1px rgba(140, 90, 60, 0.14);
	}

	.sidebar-link-active::after {
		color: var(--color-ink);
	}

	.sidebar-link-secondary {
		background: #fcfaf6;
	}

	.sidebar-link-active.sidebar-link-secondary {
		background: linear-gradient(180deg, rgba(240, 224, 206, 0.96) 0%, rgba(233, 217, 199, 0.92) 100%);
	}

	.sidebar-footer {
		display: flex;
		align-items: flex-end;
		justify-content: space-between;
		gap: 16px;
		padding-top: 16px;
		border-top: 1px solid var(--color-border);
	}

	.sidebar-user-label,
	.sidebar-user-name {
		margin: 0;
	}

	.sidebar-user-label {
		font-size: 12px;
		color: var(--color-muted);
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}

	.sidebar-user-name {
		margin-top: 6px;
		font-size: 16px;
		font-weight: 600;
	}

	.app-content {
		min-width: 0;
		padding-top: 4px;
		padding-bottom: 32px;
	}

	.app-content-wide {
		grid-column: 1 / -1;
	}

	.page-loading {
		display: flex;
		align-items: center;
		justify-content: center;
		min-height: calc(100vh - 48px);
		color: var(--color-muted);
	}

	@media (max-width: 920px) {
		.app-shell {
			grid-template-columns: 1fr;
			padding: 16px;
			gap: 16px;
		}

		.app-sidebar {
			position: static;
			height: auto;
			padding: 16px 16px 14px;
			gap: 14px;
			border-radius: 22px;
		}

		.sidebar-brand h1 {
			font-size: 24px;
		}

		.sidebar-meta {
			margin-top: 8px;
			font-size: 13px;
			line-height: 1.6;
		}

		.sidebar-nav {
			display: grid;
			grid-template-columns: repeat(2, minmax(0, 1fr));
			gap: 10px;
		}

		.sidebar-link {
			padding: 12px 14px;
			font-size: 14px;
			border-radius: 14px;
		}

		.sidebar-footer {
			align-items: center;
			padding-top: 12px;
		}
	}

	@media (max-width: 640px) {
		.app-shell {
			padding: 12px;
		}

		.app-sidebar {
			padding: 14px;
			border-radius: 20px;
		}

		.sidebar-brand {
			display: grid;
			gap: 4px;
		}

		.sidebar-brand h1 {
			font-size: 22px;
		}

		.sidebar-kicker {
			margin-bottom: 4px;
		}

		.sidebar-meta {
			display: none;
		}

		.sidebar-footer {
			padding-top: 10px;
		}
	}
</style>
