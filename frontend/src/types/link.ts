export namespace models {
    export class DocumentLink {
        id: string;
        sourceType: string;
        sourceId: string;
        targetType: string;
        targetId: string;
        linkType: string;
        createdBy: string;
        // Go type: time
        createdAt: any;

        sourceNumber?: string;
        targetNumber?: string;
        targetSubject?: string;

        static createFrom(source: any = {}) {
            return new DocumentLink(source);
        }

        constructor(source: any = {}) {
            if ('string' === typeof source) source = JSON.parse(source);
            this.id = source["id"];
            this.sourceType = source["sourceType"];
            this.sourceId = source["sourceId"];
            this.targetType = source["targetType"];
            this.targetId = source["targetId"];
            this.linkType = source["linkType"];
            this.createdBy = source["createdBy"];
            this.createdAt = source["createdAt"];
            this.sourceNumber = source["sourceNumber"];
            this.targetNumber = source["targetNumber"];
            this.targetSubject = source["targetSubject"];
        }
    }
}

export namespace services {
    export class GraphNode {
        id: string;
        label: string;
        type: string;
        subject: string;
        date: string;
        sender: string;
        recipient: string;

        constructor(source: any = {}) {
            if ('string' === typeof source) source = JSON.parse(source);
            this.id = source["id"];
            this.label = source["label"];
            this.type = source["type"];
            this.subject = source["subject"];
            this.date = source["date"];
            this.sender = source["sender"];
            this.recipient = source["recipient"];
        }
    }

    export class GraphEdge {
        id: string;
        source: string;
        target: string;
        label: string;

        constructor(source: any = {}) {
            if ('string' === typeof source) source = JSON.parse(source);
            this.id = source["id"];
            this.source = source["source"];
            this.target = source["target"];
            this.label = source["label"];
        }
    }

    export class GraphData {
        nodes: GraphNode[];
        edges: GraphEdge[];

        constructor(source: any = {}) {
            if ('string' === typeof source) source = JSON.parse(source);
            this.nodes = source["nodes"];
            this.edges = source["edges"];
        }
    }
}

export const LinkDocuments = (sourceID: string, targetID: string, sourceType: string, targetType: string, linkType: string): Promise<models.DocumentLink> => {
    return (window as any)['go']['services']['LinkService']['LinkDocuments'](sourceID, targetID, sourceType, targetType, linkType);
};

export const UnlinkDocument = (id: string): Promise<void> => {
    return (window as any)['go']['services']['LinkService']['UnlinkDocument'](id);
};

export const GetDocumentLinks = (docID: string): Promise<models.DocumentLink[]> => {
    return (window as any)['go']['services']['LinkService']['GetDocumentLinks'](docID);
};

export const GetDocumentFlow = (rootID: string): Promise<services.GraphData> => {
    return (window as any)['go']['services']['LinkService']['GetDocumentFlow'](rootID);
};
