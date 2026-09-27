export type AssignmentMode = 'execution' | 'control';
export type AssignmentMetric = 'new' | 'in_progress' | 'overdue' | 'due_soon' | 'acceptance';

export type AssignmentNavigation = {
    requestId: number;
    mode: AssignmentMode;
    metric?: AssignmentMetric;
};
