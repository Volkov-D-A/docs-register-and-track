export const workspaceGreeting = (date: Date): string => {
    const hour = date.getHours();
    if (hour < 5) return 'Доброй ночи!';
    if (hour < 12) return 'Доброе утро!';
    if (hour < 18) return 'Добрый день!';
    return 'Добрый вечер!';
};

const russianDate = new Intl.DateTimeFormat('ru-RU', {
    weekday: 'long', day: 'numeric', month: 'long', year: 'numeric',
});

export const workspaceDate = (date: Date): string => {
    const parts = russianDate.formatToParts(date);
    const value = (type: Intl.DateTimeFormatPartTypes) => parts.find((part) => part.type === type)?.value || '';
    const weekday = value('weekday');
    return `${weekday.charAt(0).toUpperCase()}${weekday.slice(1)}, ${value('day')} ${value('month')} ${value('year')}`;
};
