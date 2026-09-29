import dayjs from 'dayjs';

export type AssignmentType = 'execution' | 'acknowledgment' | 'all';
export type AssignmentMode = 'execution' | 'control';
export type AssignmentMetric = 'new' | 'in_progress' | 'overdue' | 'due_soon' | 'acceptance' | 'returned';

export type AssignmentNavigation = {
    requestId: number;
    mode: AssignmentMode;
    metric?: AssignmentMetric;
    type?: AssignmentType;
};


export const assignmentFiltersFromMetric = (metric?: AssignmentMetric) => ({
    status: metric === 'new' ? 'new'
        : metric === 'in_progress' ? 'in_progress'
        : metric === 'returned' ? 'returned'
        : metric === 'acceptance' ? 'completed' : '',
    dateFrom: metric === 'due_soon' ? dayjs().format('YYYY-MM-DD') : '',
    dateTo: metric === 'due_soon' ? dayjs().add(3, 'day').format('YYYY-MM-DD') : '',
    overdueOnly: metric === 'overdue',
});
