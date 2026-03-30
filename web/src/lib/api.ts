const API_BASE = '/api';

interface LoginRequest {
	username: string;
	password: string;
}

interface LoginResponse {
	success: boolean;
	username: string;
	needs_2fa?: boolean;
	challenge_token?: string;
}

interface LoginStep1Request {
	username: string;
	password: string;
}

interface LoginStep2Request {
	code: string;
	challenge_token: string;
}

interface AuthStatus {
	is_logged_in: boolean;
	username?: string;
}

interface Diary {
	id: number;
	content: string;
	create_date: string;
	created_at: string;
	updated_at: string;
}

interface DiaryListResponse {
	diaries: Diary[];
	total: number;
}

interface DiaryStats {
	total_count: number;
	max_consecutive_days: number;
	start_date: string;
	end_date: string;
	time_span_days: number;
}

type GoalRange = 'week' | 'month' | 'quarter' | 'year' | 'all';

interface Goal {
	id: number;
	name: string;
	description: string;
	unit: string;
	annual_target: number | null;
	weekly_target: number | null;
	is_active: boolean;
	sort_order: number;
	created_at: string;
	updated_at: string;
}

interface GoalRecordPayload {
	record_date: string;
	is_completed: boolean;
	quantity: number | null;
}

interface GoalDashboardItem {
	id: number;
	name: string;
	description: string;
	unit: string;
	annual_target: number | null;
	weekly_target: number | null;
	range_completed_count: number;
	range_quantity_total: number;
	annual_completed_count: number;
	annual_quantity_total: number;
	annual_remaining_value: number | null;
	annual_progress_percent: number | null;
	current_week_completed_count: number;
	current_week_progress_percent: number | null;
	checkin_record: GoalRecordPayload | null;
}

interface GoalCalendarDay {
	date: string;
	completed_goals: number;
	total_goals: number;
	intensity: number;
}

interface GoalCalendarDayRecord {
	goal_id: number;
	name: string;
	unit: string;
	is_completed: boolean;
	quantity: number | null;
}

interface GoalCalendarDayDetail {
	date: string;
	completed_goals: number;
	total_goals: number;
	items: GoalCalendarDayRecord[];
}

interface GoalPeriodDetail {
	date: string;
	completed_goals: number;
	items: GoalCalendarDayRecord[];
}

interface GoalDashboard {
	anchor_date: string;
	range: GoalRange;
	range_start_date: string;
	range_end_date: string;
	checkin_date: string;
	today_completed_count: number;
	annual_checkin_total: number;
	calendar_month: string;
	goals: GoalDashboardItem[];
	inactive_goals: Goal[];
	calendar_days: GoalCalendarDay[];
	day_details: GoalCalendarDayDetail[];
	week_details: GoalPeriodDetail[];
	month_details: GoalPeriodDetail[];
}

interface GoalYearSummary {
	year: number;
	has_records: boolean;
	recorded_goal_count: number;
	total_checkins: number;
	recorded_days: number;
	start_date: string;
	end_date: string;
}

async function fetchAPI<T>(path: string, options?: RequestInit): Promise<T> {
	const response = await fetch(`${API_BASE}${path}`, {
		credentials: 'include',
		headers: {
			'Content-Type': 'application/json',
			...options?.headers
		},
		...options
	});

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: '请求失败' }));
		throw new Error(error.error || `HTTP ${response.status}`);
	}

	return response.json();
}

export const authAPI = {
	login: (data: LoginRequest) =>
		fetchAPI<LoginResponse>('/auth/login', {
			method: 'POST',
			body: JSON.stringify(data)
		}),

	loginStep1: (data: LoginStep1Request) =>
		fetchAPI<LoginResponse>('/auth/login-step1', {
			method: 'POST',
			body: JSON.stringify(data)
		}),

	loginStep2: (data: LoginStep2Request) =>
		fetchAPI<LoginResponse>('/auth/login-step2', {
			method: 'POST',
			body: JSON.stringify(data)
		}),

	logout: () =>
		fetchAPI<{ success: boolean }>('/auth/logout', {
			method: 'POST'
		}),

	status: () =>
		fetchAPI<AuthStatus>('/auth/status')
};

export const diaryAPI = {
	list: (params?: { page?: number; per_page?: number; search?: string; start_date?: string; end_date?: string }) => {
		const query = new URLSearchParams();
		const trimmedSearch = params?.search?.trim();
		if (params?.page) query.set('page', params.page.toString());
		if (params?.per_page) query.set('per_page', params.per_page.toString());
		if (trimmedSearch) query.set('search', trimmedSearch);
		if (params?.start_date) query.set('start_date', params.start_date);
		if (params?.end_date) query.set('end_date', params.end_date);

		return fetchAPI<DiaryListResponse>(`/diary?${query}`);
	},

	get: (id: number) =>
		fetchAPI<Diary>(`/diary/${id}`),

	create: (data: { content: string; create_date: string }) =>
		fetchAPI<Diary>('/diary', {
			method: 'POST',
			body: JSON.stringify(data)
		}),

	update: (id: number, data: { content?: string; create_date?: string }) =>
		fetchAPI<Diary>(`/diary/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data)
		}),

	delete: (id: number) =>
		fetchAPI<{ success: boolean }>(`/diary/${id}`, {
			method: 'DELETE'
		}),

	stats: () =>
		fetchAPI<DiaryStats>('/diary/stats')
};

export const goalsAPI = {
	list: () =>
		fetchAPI<{ goals: Goal[] }>('/goals'),

	create: (data: { reactivate_id?: number; name: string; description?: string; unit?: string; annual_target?: number; weekly_target?: number }) =>
		fetchAPI<Goal>('/goals', {
			method: 'POST',
			body: JSON.stringify(data)
		}),

	update: (id: number, data: { name?: string; description?: string; unit?: string; annual_target?: number; weekly_target?: number }) =>
		fetchAPI<Goal>(`/goals/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data)
		}),

	delete: (id: number) =>
		fetchAPI<{ success: boolean }>(`/goals/${id}`, {
			method: 'DELETE'
		}),

	dashboard: (params?: { range?: GoalRange; date?: string; checkin_date?: string; month?: string }) => {
		const query = new URLSearchParams();
		if (params?.range) query.set('range', params.range);
		if (params?.date) query.set('date', params.date);
		if (params?.checkin_date) query.set('checkin_date', params.checkin_date);
		if (params?.month) query.set('month', params.month);

		return fetchAPI<GoalDashboard>(`/goals/dashboard?${query}`);
	},

	yearSummary: (year?: number) => {
		const query = new URLSearchParams();
		if (year) query.set('year', year.toString());
		return fetchAPI<GoalYearSummary>(`/goals/year-summary?${query}`);
	},

	upsertRecord: (id: number, date: string, data?: { quantity?: number }) =>
		fetchAPI<GoalRecordPayload>(`/goals/${id}/records/${date}`, {
			method: 'PUT',
			body: JSON.stringify(data ?? {})
		}),

	deleteRecord: (id: number, date: string) =>
		fetchAPI<{ success: boolean }>(`/goals/${id}/records/${date}`, {
			method: 'DELETE'
		})
};

export type {
	Diary,
	DiaryListResponse,
	DiaryStats,
	AuthStatus,
	LoginResponse,
	Goal,
	GoalRange,
	GoalDashboard,
	GoalDashboardItem,
	GoalCalendarDay,
	GoalCalendarDayDetail,
	GoalPeriodDetail,
	GoalYearSummary,
	GoalRecordPayload
};
