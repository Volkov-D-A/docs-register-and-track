import { useEffect, useState } from 'react';
import { addCalendarDays, organizationToday } from '../features/deadlines/deadlineEvents';

// Organization dates use Asia/Yekaterinburg (UTC+05:00, without DST).
export function useOrganizationDay() {
    const [day, setDay] = useState(organizationToday);
    useEffect(() => {
        let timer: ReturnType<typeof setTimeout>;
        const update = () => {
            const now = new Date();
            const today = organizationToday(now);
            setDay(today);
            clearTimeout(timer);
            const tomorrow = new Date(`${addCalendarDays(today, 1)}T00:00:00+05:00`);
            timer = setTimeout(update, Math.max(1, tomorrow.getTime() - now.getTime()));
        };
        const resume = () => { if (document.visibilityState !== 'hidden') update(); };
        update();
        window.addEventListener('focus', resume);
        document.addEventListener('visibilitychange', resume);
        return () => {
            clearTimeout(timer);
            window.removeEventListener('focus', resume);
            document.removeEventListener('visibilitychange', resume);
        };
    }, []);
    return day;
}
