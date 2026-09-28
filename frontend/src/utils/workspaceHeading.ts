type GreetingUser = {
    firstName: string;
    patronymic: string;
    noPatronymic: boolean;
};

export const workspaceGreeting = (date: Date, user?: GreetingUser | null): string => {
    const hour = date.getHours();
    const greeting = hour < 5 ? 'Доброй ночи'
        : hour < 12 ? 'Доброе утро'
        : hour < 18 ? 'Добрый день' : 'Добрый вечер';
    const name = [user?.firstName.trim(), user?.noPatronymic ? '' : user?.patronymic.trim()]
        .filter(Boolean).join(' ');
    return name ? `${greeting}, ${name}!` : `${greeting}!`;
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
