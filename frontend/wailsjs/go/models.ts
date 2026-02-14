export namespace models {
	
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
	    incomingCountMonth?: number;
	    outgoingCountMonth?: number;
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
	        this.incomingCountMonth = source["incomingCountMonth"];
	        this.outgoingCountMonth = source["outgoingCountMonth"];
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
	export class PagedResult {
	    items: any;
	    totalCount: number;
	    page: number;
	    pageSize: number;
	
	    static createFrom(source: any = {}) {
	        return new PagedResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = source["items"];
	        this.totalCount = source["totalCount"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
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

}

