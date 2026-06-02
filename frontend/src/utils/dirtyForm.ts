type FormLike = {
    isFieldsTouched?: (allFields?: boolean) => boolean;
};

type ModalLike = {
    confirm: (config: {
        title: string;
        content: string;
        okText: string;
        cancelText: string;
        okButtonProps?: { danger?: boolean };
        onOk: () => void;
    }) => void;
};

export const hasUnsavedFormChanges = (form: FormLike): boolean => (
    typeof form?.isFieldsTouched === 'function' && form.isFieldsTouched(true)
);

export const confirmDiscardFormChanges = (
    modal: ModalLike,
    form: FormLike,
    onDiscard: () => void,
) => {
    if (!hasUnsavedFormChanges(form)) {
        onDiscard();
        return;
    }

    modal.confirm({
        title: 'Закрыть без сохранения?',
        content: 'Внесенные изменения будут потеряны.',
        okText: 'Закрыть',
        cancelText: 'Продолжить редактирование',
        okButtonProps: { danger: true },
        onOk: onDiscard,
    });
};
