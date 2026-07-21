import React from 'react';
import DocumentListPageHeader from './DocumentListPageHeader';
import DocumentFilterPanel from './DocumentFilterPanel';
import DocumentListTable from './DocumentListTable';
import DocumentViewModal from './DocumentViewModal';

type DocumentListPageProps = {
    title: string;
    nomenclatureFilter?: React.ReactNode;
    onSearch: (value: string) => void;
    canRegister: boolean;
    onRegister: () => void;
    hasFilters: boolean;
    filtersContent: React.ReactNode;
    tableClassName: string;
    columns: any[];
    data: any[];
    loading: boolean;
    page: number;
    pageSize: number;
    hasMore: boolean;
    canGoBack: boolean;
    onPreviousPage: () => void;
    onNextPage: () => void;
    onPageSizeChange: (pageSize: number) => void;
    viewModalOpen: boolean;
    onCloseViewModal: () => void;
    viewDocId: string;
    documentKind: string;
};

const DocumentListPage: React.FC<DocumentListPageProps> = ({
    title,
    nomenclatureFilter,
    onSearch,
    canRegister,
    onRegister,
    hasFilters,
    filtersContent,
    tableClassName,
    columns,
    data,
    loading,
    page,
    pageSize,
    hasMore,
    canGoBack,
    onPreviousPage,
    onNextPage,
    onPageSizeChange,
    viewModalOpen,
    onCloseViewModal,
    viewDocId,
    documentKind,
}) => (
    <>
        <DocumentListPageHeader
            title={title}
            nomenclatureFilter={nomenclatureFilter}
            onSearch={onSearch}
            canRegister={canRegister}
            onRegister={onRegister}
        />

        <DocumentFilterPanel hasFilters={hasFilters}>
            {filtersContent}
        </DocumentFilterPanel>

        <DocumentListTable
            className={tableClassName}
            columns={columns}
            data={data}
            loading={loading}
            page={page}
            pageSize={pageSize}
            hasMore={hasMore}
            canGoBack={canGoBack}
            onPreviousPage={onPreviousPage}
            onNextPage={onNextPage}
            onPageSizeChange={onPageSizeChange}
        />

        <DocumentViewModal
            open={viewModalOpen}
            onCancel={onCloseViewModal}
            documentId={viewDocId}
            documentKind={documentKind}
        />
    </>
);

export default DocumentListPage;
