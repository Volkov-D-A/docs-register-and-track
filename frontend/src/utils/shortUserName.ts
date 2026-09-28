type UserNameParts = {
    lastName: string;
    firstName: string;
    patronymic: string;
    noPatronymic: boolean;
};

const initial = (name: string): string => {
    const letter = Array.from(name.trim())[0];
    return letter ? `${letter.toUpperCase()}.` : '';
};

export const shortUserName = (user: UserNameParts): string => {
    const initials = initial(user.firstName)
        + (user.noPatronymic ? '' : initial(user.patronymic));
    return [user.lastName.trim(), initials].filter(Boolean).join(' ');
};
