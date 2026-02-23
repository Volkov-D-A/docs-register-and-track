export namespace database {
	
	export class MigrationStatus {
	    currentVersion: number;
	    dirty: boolean;
	    totalAvailable: number;
	    upToDate: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MigrationStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentVersion = source["currentVersion"];
	        this.dirty = source["dirty"];
	        this.totalAvailable = source["totalAvailable"];
	        this.upToDate = source["upToDate"];
	    }
	}

}

export namespace models {
	
	export class AcknowledgmentUser {
	    id: string;
	    userId: string;
	    userName?: string;
	    // Go type: time
	    viewedAt?: any;
	    // Go type: time
	    confirmedAt?: any;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new AcknowledgmentUser(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.userId = source["userId"];
	        this.userName = source["userName"];
	        this.viewedAt = this.convertValues(source["viewedAt"], null);
	        this.confirmedAt = this.convertValues(source["confirmedAt"], null);
	        this.createdAt = this.convertValues(source["createdAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Acknowledgment {
	    id: string;
	    documentId: string;
	    documentType: string;
	    documentNumber?: string;
	    creatorId: string;
	    creatorName?: string;
	    content: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    completedAt?: any;
	    users?: AcknowledgmentUser[];
	    userIds?: string[];
	
	    static createFrom(source: any = {}) {
	        return new Acknowledgment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.documentId = source["documentId"];
	        this.documentType = source["documentType"];
	        this.documentNumber = source["documentNumber"];
	        this.creatorId = source["creatorId"];
	        this.creatorName = source["creatorName"];
	        this.content = source["content"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.completedAt = this.convertValues(source["completedAt"], null);
	        this.users = this.convertValues(source["users"], AcknowledgmentUser);
	        this.userIds = source["userIds"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class Nomenclature {
	    id: string;
	    name: string;
	    index: string;
	    year: number;
	    direction: string;
	    nextNumber: number;
	    isActive: boolean;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Nomenclature(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.index = source["index"];
	        this.year = source["year"];
	        this.direction = source["direction"];
	        this.nextNumber = source["nextNumber"];
	        this.isActive = source["isActive"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Department {
	    id: string;
	    name: string;
	    nomenclatureIds: string[];
	    nomenclature: Nomenclature[];
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Department(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.nomenclatureIds = source["nomenclatureIds"];
	        this.nomenclature = this.convertValues(source["nomenclature"], Nomenclature);
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class User {
	    id: string;
	    login: string;
	    fullName: string;
	    isActive: boolean;
	    roles: string[];
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    department?: Department;
	
	    static createFrom(source: any = {}) {
	        return new User(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.login = source["login"];
	        this.fullName = source["fullName"];
	        this.isActive = source["isActive"];
	        this.roles = source["roles"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.department = this.convertValues(source["department"], Department);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Assignment {
	    id: string;
	    documentId: string;
	    documentType: string;
	    executorId: string;
	    executorName?: string;
	    content: string;
	    // Go type: time
	    deadline?: any;
	    status: string;
	    report?: string;
	    // Go type: time
	    completedAt?: any;
	    documentNumber?: string;
	    documentSubject?: string;
	    coExecutors?: User[];
	    coExecutorIds?: string[];
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Assignment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.documentId = source["documentId"];
	        this.documentType = source["documentType"];
	        this.executorId = source["executorId"];
	        this.executorName = source["executorName"];
	        this.content = source["content"];
	        this.deadline = this.convertValues(source["deadline"], null);
	        this.status = source["status"];
	        this.report = source["report"];
	        this.completedAt = this.convertValues(source["completedAt"], null);
	        this.documentNumber = source["documentNumber"];
	        this.documentSubject = source["documentSubject"];
	        this.coExecutors = this.convertValues(source["coExecutors"], User);
	        this.coExecutorIds = source["coExecutorIds"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AssignmentFilter {
	    search?: string;
	    documentId?: string;
	    executorId?: string;
	    status?: string;
	    dateFrom?: string;
	    dateTo?: string;
	    overdueOnly: boolean;
	    showFinished: boolean;
	    page: number;
	    pageSize: number;
	
	    static createFrom(source: any = {}) {
	        return new AssignmentFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.search = source["search"];
	        this.documentId = source["documentId"];
	        this.executorId = source["executorId"];
	        this.status = source["status"];
	        this.dateFrom = source["dateFrom"];
	        this.dateTo = source["dateTo"];
	        this.overdueOnly = source["overdueOnly"];
	        this.showFinished = source["showFinished"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	    }
	}
	export class Attachment {
	    id: string;
	    documentId: string;
	    documentType: string;
	    filename: string;
	    filepath: string;
	    fileSize: number;
	    contentType: string;
	    uploadedBy: string;
	    uploadedByName?: string;
	    // Go type: time
	    uploadedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Attachment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.documentId = source["documentId"];
	        this.documentType = source["documentType"];
	        this.filename = source["filename"];
	        this.filepath = source["filepath"];
	        this.fileSize = source["fileSize"];
	        this.contentType = source["contentType"];
	        this.uploadedBy = source["uploadedBy"];
	        this.uploadedByName = source["uploadedByName"];
	        this.uploadedAt = this.convertValues(source["uploadedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CreateUserRequest {
	    login: string;
	    password: string;
	    fullName: string;
	    roles: string[];
	    departmentId: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateUserRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.login = source["login"];
	        this.password = source["password"];
	        this.fullName = source["fullName"];
	        this.roles = source["roles"];
	        this.departmentId = source["departmentId"];
	    }
	}
	export class DashboardStats {
	    role: string;
	    myAssignmentsNew?: number;
	    myAssignmentsInProgress?: number;
	    myAssignmentsOverdue?: number;
	    myAssignmentsFinished?: number;
	    myAssignmentsFinishedLate?: number;
	    incomingCount?: number;
	    outgoingCount?: number;
	    allAssignmentsOverdue?: number;
	    allAssignmentsFinished?: number;
	    allAssignmentsFinishedLate?: number;
	    userCount?: number;
	    totalDocuments?: number;
	    dbSize?: string;
	    expiringAssignments?: Assignment[];
	
	    static createFrom(source: any = {}) {
	        return new DashboardStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.myAssignmentsNew = source["myAssignmentsNew"];
	        this.myAssignmentsInProgress = source["myAssignmentsInProgress"];
	        this.myAssignmentsOverdue = source["myAssignmentsOverdue"];
	        this.myAssignmentsFinished = source["myAssignmentsFinished"];
	        this.myAssignmentsFinishedLate = source["myAssignmentsFinishedLate"];
	        this.incomingCount = source["incomingCount"];
	        this.outgoingCount = source["outgoingCount"];
	        this.allAssignmentsOverdue = source["allAssignmentsOverdue"];
	        this.allAssignmentsFinished = source["allAssignmentsFinished"];
	        this.allAssignmentsFinishedLate = source["allAssignmentsFinishedLate"];
	        this.userCount = source["userCount"];
	        this.totalDocuments = source["totalDocuments"];
	        this.dbSize = source["dbSize"];
	        this.expiringAssignments = this.convertValues(source["expiringAssignments"], Assignment);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class DocumentFilter {
	    nomenclatureId?: string;
	    nomenclatureIds?: string[];
	    documentTypeId?: string;
	    orgId?: string;
	    dateFrom?: string;
	    dateTo?: string;
	    search?: string;
	    incomingNumber?: string;
	    outgoingNumber?: string;
	    senderName?: string;
	    outgoingDateFrom?: string;
	    outgoingDateTo?: string;
	    resolution?: string;
	    noResolution?: boolean;
	    page: number;
	    pageSize: number;
	
	    static createFrom(source: any = {}) {
	        return new DocumentFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nomenclatureId = source["nomenclatureId"];
	        this.nomenclatureIds = source["nomenclatureIds"];
	        this.documentTypeId = source["documentTypeId"];
	        this.orgId = source["orgId"];
	        this.dateFrom = source["dateFrom"];
	        this.dateTo = source["dateTo"];
	        this.search = source["search"];
	        this.incomingNumber = source["incomingNumber"];
	        this.outgoingNumber = source["outgoingNumber"];
	        this.senderName = source["senderName"];
	        this.outgoingDateFrom = source["outgoingDateFrom"];
	        this.outgoingDateTo = source["outgoingDateTo"];
	        this.resolution = source["resolution"];
	        this.noResolution = source["noResolution"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	    }
	}
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
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.sourceNumber = source["sourceNumber"];
	        this.targetNumber = source["targetNumber"];
	        this.targetSubject = source["targetSubject"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DocumentType {
	    id: string;
	    name: string;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new DocumentType(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DownloadResponse {
	    filename: string;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new DownloadResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filename = source["filename"];
	        this.content = source["content"];
	    }
	}
	export class IncomingDocument {
	    id: string;
	    nomenclatureId: string;
	    nomenclatureName?: string;
	    incomingNumber: string;
	    // Go type: time
	    incomingDate: any;
	    outgoingNumberSender: string;
	    // Go type: time
	    outgoingDateSender: any;
	    intermediateNumber?: string;
	    // Go type: time
	    intermediateDate?: any;
	    documentTypeId: string;
	    documentTypeName?: string;
	    subject: string;
	    pagesCount: number;
	    content: string;
	    senderOrgId: string;
	    senderOrgName?: string;
	    senderSignatory: string;
	    senderExecutor: string;
	    recipientOrgId: string;
	    recipientOrgName?: string;
	    addressee: string;
	    resolution?: string;
	    createdBy: string;
	    createdByName?: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    attachmentsCount?: number;
	    assignmentsCount?: number;
	
	    static createFrom(source: any = {}) {
	        return new IncomingDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.nomenclatureId = source["nomenclatureId"];
	        this.nomenclatureName = source["nomenclatureName"];
	        this.incomingNumber = source["incomingNumber"];
	        this.incomingDate = this.convertValues(source["incomingDate"], null);
	        this.outgoingNumberSender = source["outgoingNumberSender"];
	        this.outgoingDateSender = this.convertValues(source["outgoingDateSender"], null);
	        this.intermediateNumber = source["intermediateNumber"];
	        this.intermediateDate = this.convertValues(source["intermediateDate"], null);
	        this.documentTypeId = source["documentTypeId"];
	        this.documentTypeName = source["documentTypeName"];
	        this.subject = source["subject"];
	        this.pagesCount = source["pagesCount"];
	        this.content = source["content"];
	        this.senderOrgId = source["senderOrgId"];
	        this.senderOrgName = source["senderOrgName"];
	        this.senderSignatory = source["senderSignatory"];
	        this.senderExecutor = source["senderExecutor"];
	        this.recipientOrgId = source["recipientOrgId"];
	        this.recipientOrgName = source["recipientOrgName"];
	        this.addressee = source["addressee"];
	        this.resolution = source["resolution"];
	        this.createdBy = source["createdBy"];
	        this.createdByName = source["createdByName"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.attachmentsCount = source["attachmentsCount"];
	        this.assignmentsCount = source["assignmentsCount"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class Organization {
	    id: string;
	    name: string;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Organization(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OutgoingDocument {
	    id: string;
	    nomenclatureId: string;
	    nomenclatureName?: string;
	    outgoingNumber: string;
	    // Go type: time
	    outgoingDate: any;
	    documentTypeId: string;
	    documentTypeName?: string;
	    subject: string;
	    pagesCount: number;
	    content: string;
	    senderOrgId: string;
	    senderOrgName?: string;
	    senderSignatory: string;
	    senderExecutor: string;
	    recipientOrgId: string;
	    recipientOrgName?: string;
	    addressee: string;
	    createdBy: string;
	    createdByName?: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    attachmentsCount?: number;
	
	    static createFrom(source: any = {}) {
	        return new OutgoingDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.nomenclatureId = source["nomenclatureId"];
	        this.nomenclatureName = source["nomenclatureName"];
	        this.outgoingNumber = source["outgoingNumber"];
	        this.outgoingDate = this.convertValues(source["outgoingDate"], null);
	        this.documentTypeId = source["documentTypeId"];
	        this.documentTypeName = source["documentTypeName"];
	        this.subject = source["subject"];
	        this.pagesCount = source["pagesCount"];
	        this.content = source["content"];
	        this.senderOrgId = source["senderOrgId"];
	        this.senderOrgName = source["senderOrgName"];
	        this.senderSignatory = source["senderSignatory"];
	        this.senderExecutor = source["senderExecutor"];
	        this.recipientOrgId = source["recipientOrgId"];
	        this.recipientOrgName = source["recipientOrgName"];
	        this.addressee = source["addressee"];
	        this.createdBy = source["createdBy"];
	        this.createdByName = source["createdByName"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.attachmentsCount = source["attachmentsCount"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OutgoingDocumentFilter {
	    nomenclatureIds?: string[];
	    documentTypeId?: string;
	    orgId?: string;
	    dateFrom?: string;
	    dateTo?: string;
	    search?: string;
	    outgoingNumber?: string;
	    recipientName?: string;
	    page: number;
	    pageSize: number;
	
	    static createFrom(source: any = {}) {
	        return new OutgoingDocumentFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nomenclatureIds = source["nomenclatureIds"];
	        this.documentTypeId = source["documentTypeId"];
	        this.orgId = source["orgId"];
	        this.dateFrom = source["dateFrom"];
	        this.dateTo = source["dateTo"];
	        this.search = source["search"];
	        this.outgoingNumber = source["outgoingNumber"];
	        this.recipientName = source["recipientName"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	    }
	}
	export class PagedResult_docflow_internal_models_Assignment_ {
	    items: Assignment[];
	    totalCount: number;
	    page: number;
	    pageSize: number;
	
	    static createFrom(source: any = {}) {
	        return new PagedResult_docflow_internal_models_Assignment_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], Assignment);
	        this.totalCount = source["totalCount"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PagedResult_docflow_internal_models_IncomingDocument_ {
	    items: IncomingDocument[];
	    totalCount: number;
	    page: number;
	    pageSize: number;
	
	    static createFrom(source: any = {}) {
	        return new PagedResult_docflow_internal_models_IncomingDocument_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], IncomingDocument);
	        this.totalCount = source["totalCount"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PagedResult_docflow_internal_models_OutgoingDocument_ {
	    items: OutgoingDocument[];
	    totalCount: number;
	    page: number;
	    pageSize: number;
	
	    static createFrom(source: any = {}) {
	        return new PagedResult_docflow_internal_models_OutgoingDocument_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], OutgoingDocument);
	        this.totalCount = source["totalCount"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SystemSetting {
	    key: string;
	    value: string;
	    description: string;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new SystemSetting(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	        this.description = source["description"];
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UpdateUserRequest {
	    id: string;
	    login: string;
	    fullName: string;
	    isActive: boolean;
	    roles: string[];
	    departmentId: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateUserRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.login = source["login"];
	        this.fullName = source["fullName"];
	        this.isActive = source["isActive"];
	        this.roles = source["roles"];
	        this.departmentId = source["departmentId"];
	    }
	}

}

export namespace services {
	
	export class GraphEdge {
	    id: string;
	    source: string;
	    target: string;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new GraphEdge(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.source = source["source"];
	        this.target = source["target"];
	        this.label = source["label"];
	    }
	}
	export class GraphNode {
	    id: string;
	    label: string;
	    type: string;
	    subject: string;
	    date: string;
	    sender: string;
	    recipient: string;
	
	    static createFrom(source: any = {}) {
	        return new GraphNode(source);
	    }
	
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
	export class GraphData {
	    nodes: GraphNode[];
	    edges: GraphEdge[];
	
	    static createFrom(source: any = {}) {
	        return new GraphData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nodes = this.convertValues(source["nodes"], GraphNode);
	        this.edges = this.convertValues(source["edges"], GraphEdge);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	

}

