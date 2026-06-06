export const ASSIGNMENTS_CHANGED_EVENT = 'assignments:changed';

export type AssignmentsChangedDetail = {
    documentId?: string;
};

export const emitAssignmentsChanged = (detail: AssignmentsChangedDetail = {}) => {
    window.dispatchEvent(new CustomEvent<AssignmentsChangedDetail>(ASSIGNMENTS_CHANGED_EVENT, {
        detail,
    }));
};

export const onAssignmentsChanged = (
    listener: (detail: AssignmentsChangedDetail) => void,
) => {
    const handler = (event: Event) => {
        listener((event as CustomEvent<AssignmentsChangedDetail>).detail);
    };

    window.addEventListener(ASSIGNMENTS_CHANGED_EVENT, handler);
    return () => window.removeEventListener(ASSIGNMENTS_CHANGED_EVENT, handler);
};
