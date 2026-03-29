const WEEKDAYS = ['星期日', '星期一', '星期二', '星期三', '星期四', '星期五', '星期六'];
const MONTH_NAMES = ['一月', '二月', '三月', '四月', '五月', '六月', '七月', '八月', '九月', '十月', '十一月', '十二月'];

function pad2(value: number): string {
	return value.toString().padStart(2, '0');
}

function parseDateParts(dateStr: string): [number, number, number] {
	const [year, month, day] = dateStr.split('-').map(Number);
	return [year, month, day];
}

function createLocalDate(dateStr: string): Date {
	const [year, month, day] = parseDateParts(dateStr);
	return new Date(year, month - 1, day);
}

export function getTodayDateString(): string {
	const now = new Date();
	return `${now.getFullYear()}-${pad2(now.getMonth() + 1)}-${pad2(now.getDate())}`;
}

export function getCurrentMonthString(): string {
	const now = new Date();
	return `${now.getFullYear()}-${pad2(now.getMonth() + 1)}`;
}

export function getYesterdayDateString(): string {
	const now = new Date();
	now.setDate(now.getDate() - 1);
	return `${now.getFullYear()}-${pad2(now.getMonth() + 1)}-${pad2(now.getDate())}`;
}

export function getMonthRange(monthStr: string): { startDate: string; endDate: string } {
	const [year, month] = monthStr.split('-').map(Number);
	const lastDay = new Date(year, month, 0).getDate();

	return {
		startDate: `${year}-${pad2(month)}-01`,
		endDate: `${year}-${pad2(month)}-${pad2(lastDay)}`
	};
}

export function formatMonthLabel(monthStr: string): string {
	if (!monthStr) return '';
	const [year, month] = monthStr.split('-').map(Number);
	return `${year}年${MONTH_NAMES[month - 1]}`;
}

export function formatWeekday(dateStr: string): string {
	return WEEKDAYS[createLocalDate(dateStr).getDay()];
}

export function formatLongDate(dateStr: string): string {
	const [year, month, day] = parseDateParts(dateStr);
	return `${year}年${month}月${day}日 ${formatWeekday(dateStr)}`;
}

export function shiftMonth(monthStr: string, delta: number): string {
	const [year, month] = monthStr.split('-').map(Number);
	const next = new Date(year, month - 1 + delta, 1);
	return `${next.getFullYear()}-${pad2(next.getMonth() + 1)}`;
}

export function getMonthCalendarDays(monthStr: string): Array<{ date: string; day: number; inMonth: boolean }> {
	if (!monthStr) return [];

	const [year, month] = monthStr.split('-').map(Number);
	const first = new Date(year, month - 1, 1);
	const startWeekday = (first.getDay() + 6) % 7;
	const start = new Date(year, month - 1, 1 - startWeekday);
	const cells: Array<{ date: string; day: number; inMonth: boolean }> = [];

	for (let index = 0; index < 42; index += 1) {
		const current = new Date(start);
		current.setDate(start.getDate() + index);
		cells.push({
			date: `${current.getFullYear()}-${pad2(current.getMonth() + 1)}-${pad2(current.getDate())}`,
			day: current.getDate(),
			inMonth: current.getMonth() === month - 1
		});
	}

	return cells;
}
