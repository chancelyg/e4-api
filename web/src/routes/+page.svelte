<script lang="ts">
	import { auth } from '$lib/stores.svelte';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';

	let username = $state('');
	let password = $state('');
	let totpCode = $state('');
	let error = $state('');
	let isLoading = $state(false);
	let needs2FA = $state(false);
	let challengeToken = $state('');

	onMount(async () => {
		await auth.ensureInitialized();
		if (auth.isLoggedIn) {
			goto('/diary');
		}
	});

	async function handleLogin(event: Event) {
		event.preventDefault();
		error = '';
		isLoading = true;

		try {
			const result = await auth.login(username, password);
			if (result.needs_2fa && result.challenge_token) {
				needs2FA = true;
				challengeToken = result.challenge_token;
			} else {
				goto('/diary');
			}
		} catch (err) {
			error = err instanceof Error ? err.message : '登录失败';
		} finally {
			isLoading = false;
		}
	}

	async function handle2FA(event: Event) {
		event.preventDefault();
		error = '';
		isLoading = true;

		try {
			const result = await auth.verify2FA(totpCode, challengeToken);
			if (result.success) {
				goto('/diary');
			} else {
				error = '验证码错误';
			}
		} catch (err) {
			error = err instanceof Error ? err.message : '验证失败';
		} finally {
			isLoading = false;
		}
	}

	function resetLogin() {
		needs2FA = false;
		totpCode = '';
		challengeToken = '';
		error = '';
	}
</script>

<div class="login-page">
	<section class="login-copy">
		<p class="login-kicker">E4 Diary</p>
		<h1>把每天写成一份安静、可检索的记录。</h1>
		<p class="login-text">
			登录后即可查看日记、按月份筛选内容，并通过模糊搜索快速找回过去的片段。
		</p>
		<div class="login-note">
			<p>适合单机部署的个人日记服务。</p>
			<p>默认账号：`admin / admin`</p>
		</div>
	</section>

	<section class="login-card">
		<div class="login-card-head">
			<h2>{needs2FA ? '验证身份' : '登录'}</h2>
			<p>{needs2FA ? '请输入认证器中的 6 位验证码。' : '输入账号密码进入你的日记。'}</p>
		</div>

		{#if error}
			<div class="alert variant-soft-error">
				<p>{error}</p>
			</div>
		{/if}

		{#if !needs2FA}
			<form onsubmit={handleLogin} class="login-form">
				<label class="login-field">
					<span>用户名</span>
					<input type="text" bind:value={username} placeholder="admin" required />
				</label>

				<label class="login-field">
					<span>密码</span>
					<input type="password" bind:value={password} placeholder="输入密码" required />
				</label>

				<button type="submit" class="btn login-submit" disabled={isLoading}>
					{isLoading ? '登录中...' : '进入日记'}
				</button>
			</form>
		{:else}
			<form onsubmit={handle2FA} class="login-form">
				<label class="login-field">
					<span>二步验证码</span>
					<input
						type="text"
						bind:value={totpCode}
						placeholder="输入 6 位验证码"
						maxlength="6"
						pattern="[0-9]*"
						inputmode="numeric"
						required
					/>
				</label>

				<button type="submit" class="btn login-submit" disabled={isLoading}>
					{isLoading ? '验证中...' : '确认登录'}
				</button>
				<button type="button" class="btn variant-soft-surface" onclick={resetLogin}>返回上一步</button>
			</form>
		{/if}
	</section>
</div>

<style>
	.login-page {
		min-height: 100vh;
		display: grid;
		grid-template-columns: minmax(0, 1.1fr) minmax(320px, 420px);
		align-items: center;
		gap: 48px;
		max-width: 1180px;
		margin: 0 auto;
		padding: 32px 24px;
	}

	.login-copy {
		padding: 32px 12px;
	}

	.login-kicker {
		margin: 0 0 12px;
		font-size: 12px;
		letter-spacing: 0.16em;
		text-transform: uppercase;
		color: var(--color-muted);
	}

	.login-copy h1 {
		margin: 0;
		max-width: 10ch;
		font-family: var(--font-family-display);
		font-size: clamp(2.5rem, 5vw, 4.6rem);
		font-weight: 600;
		line-height: 1.04;
	}

	.login-text {
		max-width: 36rem;
		margin: 20px 0 0;
		font-size: 1rem;
		line-height: 1.9;
		color: var(--color-ink-soft);
	}

	.login-note {
		margin-top: 24px;
		padding-top: 18px;
		border-top: 1px solid var(--color-border);
		color: var(--color-muted);
		font-size: 0.95rem;
		line-height: 1.8;
	}

	.login-note p {
		margin: 0;
	}

	.login-card {
		padding: 28px;
		border-radius: 28px;
		border: 1px solid var(--color-border);
		background: var(--color-panel);
		box-shadow: var(--shadow-soft);
	}

	.login-card-head h2 {
		margin: 0;
		font-size: 1.5rem;
	}

	.login-card-head p {
		margin: 8px 0 0;
		font-size: 0.95rem;
		line-height: 1.7;
		color: var(--color-muted);
	}

	.login-form {
		display: grid;
		gap: 18px;
		margin-top: 22px;
	}

	.login-field {
		display: grid;
		gap: 8px;
	}

	.login-field span {
		font-size: 0.92rem;
		color: var(--color-ink-soft);
	}

	.login-field input {
		min-height: 48px;
		padding: 0 14px;
		border: 1px solid var(--color-border);
		border-radius: 14px;
		background: rgba(255, 255, 255, 0.85);
	}

	.login-field input:focus {
		outline: none;
		border-color: rgba(140, 90, 60, 0.45);
		box-shadow: 0 0 0 4px rgba(140, 90, 60, 0.08);
	}

	.login-submit {
		width: 100%;
		background: var(--color-accent);
		color: white;
	}

	.login-submit:hover:not(:disabled) {
		background: var(--color-accent-strong);
	}

	@media (max-width: 860px) {
		.login-page {
			grid-template-columns: 1fr;
			gap: 20px;
			padding: 20px 16px 32px;
		}

		.login-copy {
			padding: 8px 0;
		}

		.login-copy h1 {
			max-width: none;
			font-size: 2.6rem;
		}
	}
</style>
