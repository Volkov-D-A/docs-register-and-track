import React from 'react';
import { Modal, Select, Tag } from 'antd';
import DocumentListPage from './DocumentListPage';

type DocumentKindPageProps = {
    title: string;
    filterDisabled: boolean;
    nomenclatures: any[];
    filterNomenclatureIds: string[];
    setFilterNomenclatureIds: (values: string[]) => void;
    setPage: (page: number) => void;
    onSearch: (value: string) => void;
    canRegister: boolean;
    onOpenRegister: () => void;
    hasFilters: boolean;
    filtersContent: React.ReactNode;
    tableClassName: string;
    columns: any[];
    data: any[];
    loading: boolean;
    page: number;
    pageSize: number;
    totalCount: number;
    onPageChange: (page: number, pageSize: number) => void;
    viewModalOpen: boolean;
    onCloseViewModal: () => void;
    viewDocId: string;
    documentKind: string;
    registerModal: {
        title: string;
        open: boolean;
        onCancel: () => void;
        onOk: () => void;
        width: number;
        okText?: string;
        confirmLoading?: boolean;
        linkedBadge?: React.ReactNode;
        content: React.ReactNode;
    };
    editModal: {
        title: string;
        open: boolean;
        onCancel: () => void;
        onOk: () => void;
        width: number;
        okText?: string;
        confirmLoading?: boolean;
        content: React.ReactNode;
    };
};

const DocumentKindPage: React.FC<DocumentKindPageProps> = ({
    title,
    filterDisabled,
    nomenclatures,
    filterNomenclatureIds,
    setFilterNomenclatureIds,
    setPage,
    onSearch,
    canRegister,
    onOpenRegister,
    hasFilters,
    filtersContent,
    tableClassName,
    columns,
    data,
    loading,
    page,
    pageSize,
    totalCount,
    onPageChange,
    viewModalOpen,
    onCloseViewModal,
    viewDocId,
    documentKind,
    registerModal,
    editModal,
}) => (
    <div>
        <DocumentListPage
            title={title}
            nomenclatureFilter={!filterDisabled ? (
                <Select
                    mode="multiple"
                    size="middle"
                    style={{ minWidth: 250 }}
                    placeholder="Все дела"
                    value={filterNomenclatureIds}
                    onChange={(vals: string[]) => { setFilterNomenclatureIds(vals); setPage(1); }}
                    allowClear
                    options={nomenclatures.map((n: any) => ({ value: n.id, label: `${n.index} — ${n.name}` }))}
                />
            ) : undefined}
            onSearch={onSearch}
            canRegister={canRegister}
            onRegister={onOpenRegister}
            hasFilters={hasFilters}
            filtersContent={filtersContent}
            tableClassName={tableClassName}
            columns={columns}
            data={data}
            loading={loading}
            page={page}
            pageSize={pageSize}
            totalCount={totalCount}
            onPageChange={onPageChange}
            viewModalOpen={viewModalOpen}
            onCloseViewModal={onCloseViewModal}
            viewDocId={viewDocId}
            documentKind={documentKind}
        />

        <Modal
            title={registerModal.title}
            open={registerModal.open}
            forceRender
            onCancel={registerModal.onCancel}
            onOk={registerModal.onOk}
            width={registerModal.width}
            okText={registerModal.okText}
            confirmLoading={registerModal.confirmLoading}
        >
            {registerModal.linkedBadge}
            {registerModal.content}
        </Modal>

        <Modal
            title={editModal.title}
            open={editModal.open}
            forceRender
            onCancel={editModal.onCancel}
            onOk={editModal.onOk}
            width={editModal.width}
            okText={editModal.okText}
            confirmLoading={editModal.confirmLoading}
        >
            {editModal.content}
        </Modal>
    </div>
);

export default DocumentKindPage;
