export type AssignmentMode = 'execution' | 'control';
export type AssignmentMetric = 'new' | 'overdue' | 'due_soon' | 'acceptance';

export type AssignmentNavigation = {
    requestId: number;
    mode: AssignmentMode;
    metric?: AssignmentMetric;
};
