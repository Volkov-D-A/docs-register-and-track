// Graph dates are shown as DD.MM.YYYY, while layout needs chronological order.
export const graphDateSortKey = (date: string): string => {
    if (!/^\d{2}\.\d{2}\.\d{4}$/.test(date)) return date;
    return `${date.slice(6, 10)}${date.slice(3, 5)}${date.slice(0, 2)}`;
};
