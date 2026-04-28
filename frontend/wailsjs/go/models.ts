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

export namespace dto {
	
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
	    documentKind: string;
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
	        this.documentKind = source["documentKind"];
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
	
	export class AdminAuditLog {
	    id: string;
	    userName: string;
	    action: string;
	    details: string;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new AdminAuditLog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.userName = source["userName"];
	        this.action = source["action"];
	        this.details = source["details"];
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
	export class AdminAuditLogPage {
	    items: AdminAuditLog[];
	    total: number;
	    page: number;
	
	    static createFrom(source: any = {}) {
	        return new AdminAuditLogPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], AdminAuditLog);
	        this.total = source["total"];
	        this.page = source["page"];
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
	    kindCode: string;
	    separator: string;
	    numberingMode: string;
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
	        this.kindCode = source["kindCode"];
	        this.separator = source["separator"];
	        this.numberingMode = source["numberingMode"];
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
	    isDocumentParticipant: boolean;
	    isActive: boolean;
	    failedLoginAttempts: number;
	    systemPermissions: string[];
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
	        this.isDocumentParticipant = source["isDocumentParticipant"];
	        this.isActive = source["isActive"];
	        this.failedLoginAttempts = source["failedLoginAttempts"];
	        this.systemPermissions = source["systemPermissions"];
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
	    documentKind: string;
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
	        this.documentKind = source["documentKind"];
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
	export class Attachment {
	    id: string;
	    documentId: string;
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
	export class DocumentResolution {
	    id: string;
	    resolution?: string;
	    resolutionAuthor?: string;
	    resolutionExecutors?: string;
	    position: number;
	
	    static createFrom(source: any = {}) {
	        return new DocumentResolution(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.resolution = source["resolution"];
	        this.resolutionAuthor = source["resolutionAuthor"];
	        this.resolutionExecutors = source["resolutionExecutors"];
	        this.position = source["position"];
	    }
	}
	export class DocumentCorrespondentRegistration {
	    id: string;
	    registrationNumber: string;
	    // Go type: time
	    registrationDate: any;
	    correspondentOrgId: string;
	    correspondentName?: string;
	    position: number;
	
	    static createFrom(source: any = {}) {
	        return new DocumentCorrespondentRegistration(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.registrationNumber = source["registrationNumber"];
	        this.registrationDate = this.convertValues(source["registrationDate"], null);
	        this.correspondentOrgId = source["correspondentOrgId"];
	        this.correspondentName = source["correspondentName"];
	        this.position = source["position"];
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
	export class CitizenAppealDocument {
	    id: string;
	    nomenclatureId: string;
	    nomenclatureName?: string;
	    registrationNumber: string;
	    // Go type: time
	    registrationDate: any;
	    // Go type: time
	    appealDate: any;
	    documentTypeId: string;
	    documentTypeName?: string;
	    content: string;
	    pagesCount: number;
	    applicantFullName: string;
	    registrationAddress: string;
	    appealType: string;
	    applicantCategory: string;
	    appealPagesCount: number;
	    attachmentPagesCount: number;
	    hasEnvelope: boolean;
	    receivedFromPos: boolean;
	    correspondents?: DocumentCorrespondentRegistration[];
	    resolutions?: DocumentResolution[];
	    createdBy: string;
	    createdByName?: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    attachmentsCount?: number;
	    assignmentsCount?: number;
	
	    static createFrom(source: any = {}) {
	        return new CitizenAppealDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.nomenclatureId = source["nomenclatureId"];
	        this.nomenclatureName = source["nomenclatureName"];
	        this.registrationNumber = source["registrationNumber"];
	        this.registrationDate = this.convertValues(source["registrationDate"], null);
	        this.appealDate = this.convertValues(source["appealDate"], null);
	        this.documentTypeId = source["documentTypeId"];
	        this.documentTypeName = source["documentTypeName"];
	        this.content = source["content"];
	        this.pagesCount = source["pagesCount"];
	        this.applicantFullName = source["applicantFullName"];
	        this.registrationAddress = source["registrationAddress"];
	        this.appealType = source["appealType"];
	        this.applicantCategory = source["applicantCategory"];
	        this.appealPagesCount = source["appealPagesCount"];
	        this.attachmentPagesCount = source["attachmentPagesCount"];
	        this.hasEnvelope = source["hasEnvelope"];
	        this.receivedFromPos = source["receivedFromPos"];
	        this.correspondents = this.convertValues(source["correspondents"], DocumentCorrespondentRegistration);
	        this.resolutions = this.convertValues(source["resolutions"], DocumentResolution);
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
	export class DashboardActivity {
	    expiringAssignments?: Assignment[];
	
	    static createFrom(source: any = {}) {
	        return new DashboardActivity(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
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
	
	export class OutgoingDocument {
	    id: string;
	    nomenclatureId: string;
	    nomenclatureName?: string;
	    outgoingNumber: string;
	    // Go type: time
	    outgoingDate: any;
	    documentTypeId: string;
	    documentTypeName?: string;
	    content: string;
	    pagesCount: number;
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
	        this.content = source["content"];
	        this.pagesCount = source["pagesCount"];
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
	export class IncomingDocument {
	    id: string;
	    nomenclatureId: string;
	    nomenclatureName?: string;
	    incomingNumber: string;
	    // Go type: time
	    incomingDate: any;
	    documentTypeId: string;
	    documentTypeName?: string;
	    content: string;
	    pagesCount: number;
	    correspondents?: DocumentCorrespondentRegistration[];
	    senderSignatory: string;
	    resolution?: string;
	    resolutionAuthor?: string;
	    resolutionExecutors?: string;
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
	        this.documentTypeId = source["documentTypeId"];
	        this.documentTypeName = source["documentTypeName"];
	        this.content = source["content"];
	        this.pagesCount = source["pagesCount"];
	        this.correspondents = this.convertValues(source["correspondents"], DocumentCorrespondentRegistration);
	        this.senderSignatory = source["senderSignatory"];
	        this.resolution = source["resolution"];
	        this.resolutionAuthor = source["resolutionAuthor"];
	        this.resolutionExecutors = source["resolutionExecutors"];
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
	export class DocumentCard {
	    id: string;
	    kindCode: string;
	    kindName: string;
	    registrationNumber: string;
	    // Go type: time
	    registrationDate: any;
	    nomenclatureId: string;
	    nomenclatureName?: string;
	    documentTypeId: string;
	    documentTypeName?: string;
	    content: string;
	    createdBy: string;
	    createdByName?: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    incomingLetter?: IncomingDocument;
	    outgoingLetter?: OutgoingDocument;
	    citizenAppeal?: CitizenAppealDocument;
	
	    static createFrom(source: any = {}) {
	        return new DocumentCard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kindCode = source["kindCode"];
	        this.kindName = source["kindName"];
	        this.registrationNumber = source["registrationNumber"];
	        this.registrationDate = this.convertValues(source["registrationDate"], null);
	        this.nomenclatureId = source["nomenclatureId"];
	        this.nomenclatureName = source["nomenclatureName"];
	        this.documentTypeId = source["documentTypeId"];
	        this.documentTypeName = source["documentTypeName"];
	        this.content = source["content"];
	        this.createdBy = source["createdBy"];
	        this.createdByName = source["createdByName"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.incomingLetter = this.convertValues(source["incomingLetter"], IncomingDocument);
	        this.outgoingLetter = this.convertValues(source["outgoingLetter"], OutgoingDocument);
	        this.citizenAppeal = this.convertValues(source["citizenAppeal"], CitizenAppealDocument);
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
	
	export class DocumentKind {
	    code: string;
	    name: string;
	    registrationFormCode: string;
	    registryGroup: string;
	    supportedActions: string[];
	    availableActions: string[];
	
	    static createFrom(source: any = {}) {
	        return new DocumentKind(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.name = source["name"];
	        this.registrationFormCode = source["registrationFormCode"];
	        this.registryGroup = source["registryGroup"];
	        this.supportedActions = source["supportedActions"];
	        this.availableActions = source["availableActions"];
	    }
	}
	export class DocumentLink {
	    id: string;
	    sourceKind: string;
	    sourceId: string;
	    targetKind: string;
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
	        this.sourceKind = source["sourceKind"];
	        this.sourceId = source["sourceId"];
	        this.targetKind = source["targetKind"];
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
	export class DocumentListItem {
	    id: string;
	    kindCode: string;
	    kindName: string;
	    registrationNumber: string;
	    // Go type: time
	    registrationDate: any;
	    nomenclatureId: string;
	    nomenclatureName?: string;
	    documentTypeId: string;
	    documentTypeName?: string;
	    content: string;
	    pagesCount: number;
	    createdBy: string;
	    createdByName?: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    incomingNumber?: string;
	    // Go type: time
	    incomingDate?: any;
	    // Go type: time
	    appealDate?: any;
	    outgoingNumber?: string;
	    // Go type: time
	    outgoingDate?: any;
	    correspondents?: DocumentCorrespondentRegistration[];
	    senderSignatory?: string;
	    resolution?: string;
	    resolutionAuthor?: string;
	    resolutionExecutors?: string;
	    resolutions?: DocumentResolution[];
	    recipientOrgName?: string;
	    addressee?: string;
	    senderExecutor?: string;
	    applicantFullName?: string;
	    registrationAddress?: string;
	    appealType?: string;
	    applicantCategory?: string;
	    appealPagesCount?: number;
	    attachmentPagesCount?: number;
	    hasEnvelope?: boolean;
	    receivedFromPos?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DocumentListItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kindCode = source["kindCode"];
	        this.kindName = source["kindName"];
	        this.registrationNumber = source["registrationNumber"];
	        this.registrationDate = this.convertValues(source["registrationDate"], null);
	        this.nomenclatureId = source["nomenclatureId"];
	        this.nomenclatureName = source["nomenclatureName"];
	        this.documentTypeId = source["documentTypeId"];
	        this.documentTypeName = source["documentTypeName"];
	        this.content = source["content"];
	        this.pagesCount = source["pagesCount"];
	        this.createdBy = source["createdBy"];
	        this.createdByName = source["createdByName"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.incomingNumber = source["incomingNumber"];
	        this.incomingDate = this.convertValues(source["incomingDate"], null);
	        this.appealDate = this.convertValues(source["appealDate"], null);
	        this.outgoingNumber = source["outgoingNumber"];
	        this.outgoingDate = this.convertValues(source["outgoingDate"], null);
	        this.correspondents = this.convertValues(source["correspondents"], DocumentCorrespondentRegistration);
	        this.senderSignatory = source["senderSignatory"];
	        this.resolution = source["resolution"];
	        this.resolutionAuthor = source["resolutionAuthor"];
	        this.resolutionExecutors = source["resolutionExecutors"];
	        this.resolutions = this.convertValues(source["resolutions"], DocumentResolution);
	        this.recipientOrgName = source["recipientOrgName"];
	        this.addressee = source["addressee"];
	        this.senderExecutor = source["senderExecutor"];
	        this.applicantFullName = source["applicantFullName"];
	        this.registrationAddress = source["registrationAddress"];
	        this.appealType = source["appealType"];
	        this.applicantCategory = source["applicantCategory"];
	        this.appealPagesCount = source["appealPagesCount"];
	        this.attachmentPagesCount = source["attachmentPagesCount"];
	        this.hasEnvelope = source["hasEnvelope"];
	        this.receivedFromPos = source["receivedFromPos"];
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
	
	export class JournalEntry {
	    id: string;
	    documentId: string;
	    userName?: string;
	    action: string;
	    details: string;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new JournalEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.documentId = source["documentId"];
	        this.userName = source["userName"];
	        this.action = source["action"];
	        this.details = source["details"];
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
	
	export class PagedResult_github_com_Volkov_D_A_docs_register_and_track_internal_dto_Assignment_ {
	    items: Assignment[];
	    totalCount: number;
	    page: number;
	    pageSize: number;
	
	    static createFrom(source: any = {}) {
	        return new PagedResult_github_com_Volkov_D_A_docs_register_and_track_internal_dto_Assignment_(source);
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
	export class PagedResult_github_com_Volkov_D_A_docs_register_and_track_internal_dto_DocumentListItem_ {
	    items: DocumentListItem[];
	    totalCount: number;
	    page: number;
	    pageSize: number;
	
	    static createFrom(source: any = {}) {
	        return new PagedResult_github_com_Volkov_D_A_docs_register_and_track_internal_dto_DocumentListItem_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], DocumentListItem);
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
	export class ResolutionExecutor {
	    id: string;
	    name: string;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new ResolutionExecutor(source);
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

}

export namespace models {
	
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
	export class AssignmentMonthlyPoint {
	    month: number;
	    period: string;
	    total: number;
	    overdue: number;
	
	    static createFrom(source: any = {}) {
	        return new AssignmentMonthlyPoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.month = source["month"];
	        this.period = source["period"];
	        this.total = source["total"];
	        this.overdue = source["overdue"];
	    }
	}
	export class StatisticsReportRow {
	    key: string;
	    name: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new StatisticsReportRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.name = source["name"];
	        this.count = source["count"];
	    }
	}
	export class StatisticsSeriesPoint {
	    month: number;
	    period: string;
	    categoryKey: string;
	    categoryName: string;
	    value: number;
	
	    static createFrom(source: any = {}) {
	        return new StatisticsSeriesPoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.month = source["month"];
	        this.period = source["period"];
	        this.categoryKey = source["categoryKey"];
	        this.categoryName = source["categoryName"];
	        this.value = source["value"];
	    }
	}
	export class AssignmentStatistics {
	    year: number;
	    monthlyTotals: AssignmentMonthlyPoint[];
	    monthlyByExecutor: StatisticsSeriesPoint[];
	    overdueRating: StatisticsReportRow[];
	    statusCounts: StatisticsReportRow[];
	
	    static createFrom(source: any = {}) {
	        return new AssignmentStatistics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.year = source["year"];
	        this.monthlyTotals = this.convertValues(source["monthlyTotals"], AssignmentMonthlyPoint);
	        this.monthlyByExecutor = this.convertValues(source["monthlyByExecutor"], StatisticsSeriesPoint);
	        this.overdueRating = this.convertValues(source["overdueRating"], StatisticsReportRow);
	        this.statusCounts = this.convertValues(source["statusCounts"], StatisticsReportRow);
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
	export class StatisticsOption {
	    value: string;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new StatisticsOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = source["value"];
	        this.label = source["label"];
	    }
	}
	export class AssignmentStatisticsFilters {
	    users: StatisticsOption[];
	
	    static createFrom(source: any = {}) {
	        return new AssignmentStatisticsFilters(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.users = this.convertValues(source["users"], StatisticsOption);
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
	export class AssignmentStatisticsReport {
	    startDate: string;
	    endDate: string;
	    onlyOverdue: boolean;
	    userId?: string;
	    total: number;
	    rows: StatisticsReportRow[];
	
	    static createFrom(source: any = {}) {
	        return new AssignmentStatisticsReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.startDate = source["startDate"];
	        this.endDate = source["endDate"];
	        this.onlyOverdue = source["onlyOverdue"];
	        this.userId = source["userId"];
	        this.total = source["total"];
	        this.rows = this.convertValues(source["rows"], StatisticsReportRow);
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
	export class CreateJournalEntryRequest {
	    DocumentID: number[];
	    UserID: number[];
	    Action: string;
	    Details: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateJournalEntryRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.DocumentID = source["DocumentID"];
	        this.UserID = source["UserID"];
	        this.Action = source["Action"];
	        this.Details = source["Details"];
	    }
	}
	export class CreateUserRequest {
	    login: string;
	    password: string;
	    fullName: string;
	    departmentId: string;
	    isDocumentParticipant: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CreateUserRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.login = source["login"];
	        this.password = source["password"];
	        this.fullName = source["fullName"];
	        this.departmentId = source["departmentId"];
	        this.isDocumentParticipant = source["isDocumentParticipant"];
	    }
	}
	export class DocumentFilter {
	    nomenclatureId?: string;
	    nomenclatureIds?: string[];
	    kindCode?: string;
	    documentTypeId?: string;
	    orgId?: string;
	    dateFrom?: string;
	    dateTo?: string;
	    search?: string;
	    incomingNumber?: string;
	    registrationNumber?: string;
	    outgoingNumber?: string;
	    recipientName?: string;
	    senderName?: string;
	    applicantName?: string;
	    appealType?: string;
	    appealDateFrom?: string;
	    appealDateTo?: string;
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
	        this.kindCode = source["kindCode"];
	        this.documentTypeId = source["documentTypeId"];
	        this.orgId = source["orgId"];
	        this.dateFrom = source["dateFrom"];
	        this.dateTo = source["dateTo"];
	        this.search = source["search"];
	        this.incomingNumber = source["incomingNumber"];
	        this.registrationNumber = source["registrationNumber"];
	        this.outgoingNumber = source["outgoingNumber"];
	        this.recipientName = source["recipientName"];
	        this.senderName = source["senderName"];
	        this.applicantName = source["applicantName"];
	        this.appealType = source["appealType"];
	        this.appealDateFrom = source["appealDateFrom"];
	        this.appealDateTo = source["appealDateTo"];
	        this.outgoingDateFrom = source["outgoingDateFrom"];
	        this.outgoingDateTo = source["outgoingDateTo"];
	        this.resolution = source["resolution"];
	        this.noResolution = source["noResolution"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	    }
	}
	export class DocumentStatistics {
	    year: number;
	    totalYear: number;
	    documentsByKindMonthly: StatisticsSeriesPoint[];
	    documentsByRegistrarMonthly: StatisticsSeriesPoint[];
	
	    static createFrom(source: any = {}) {
	        return new DocumentStatistics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.year = source["year"];
	        this.totalYear = source["totalYear"];
	        this.documentsByKindMonthly = this.convertValues(source["documentsByKindMonthly"], StatisticsSeriesPoint);
	        this.documentsByRegistrarMonthly = this.convertValues(source["documentsByRegistrarMonthly"], StatisticsSeriesPoint);
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
	export class DocumentStatisticsFilters {
	    kinds: StatisticsOption[];
	    nomenclature: StatisticsOption[];
	    users: StatisticsOption[];
	
	    static createFrom(source: any = {}) {
	        return new DocumentStatisticsFilters(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kinds = this.convertValues(source["kinds"], StatisticsOption);
	        this.nomenclature = this.convertValues(source["nomenclature"], StatisticsOption);
	        this.users = this.convertValues(source["users"], StatisticsOption);
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
	export class DocumentStatisticsReport {
	    startDate: string;
	    endDate: string;
	    groupBy: string;
	    total: number;
	    rows: StatisticsReportRow[];
	
	    static createFrom(source: any = {}) {
	        return new DocumentStatisticsReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.startDate = source["startDate"];
	        this.endDate = source["endDate"];
	        this.groupBy = source["groupBy"];
	        this.total = source["total"];
	        this.rows = this.convertValues(source["rows"], StatisticsReportRow);
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
	    kindCode: string;
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
	        this.kindCode = source["kindCode"];
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
	
	
	export class ReleaseNoteChange {
	    id: number[];
	    releaseNoteId: number[];
	    sortOrder: number;
	    title: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new ReleaseNoteChange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.releaseNoteId = source["releaseNoteId"];
	        this.sortOrder = source["sortOrder"];
	        this.title = source["title"];
	        this.description = source["description"];
	    }
	}
	export class ReleaseNote {
	    id: number[];
	    version: string;
	    // Go type: time
	    releasedAt: any;
	    isCurrent: boolean;
	    // Go type: time
	    createdAt: any;
	    changes: ReleaseNoteChange[];
	    isViewed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ReleaseNote(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.version = source["version"];
	        this.releasedAt = this.convertValues(source["releasedAt"], null);
	        this.isCurrent = source["isCurrent"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.changes = this.convertValues(source["changes"], ReleaseNoteChange);
	        this.isViewed = source["isViewed"];
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
	export class SystemStatistics {
	    userCount: number;
	    totalDocuments: number;
	    dbSize: string;
	    storageObjects: number;
	    storageSize: string;
	
	    static createFrom(source: any = {}) {
	        return new SystemStatistics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.userCount = source["userCount"];
	        this.totalDocuments = source["totalDocuments"];
	        this.dbSize = source["dbSize"];
	        this.storageObjects = source["storageObjects"];
	        this.storageSize = source["storageSize"];
	    }
	}
	export class UpdateProfileRequest {
	    login: string;
	    fullName: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateProfileRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.login = source["login"];
	        this.fullName = source["fullName"];
	    }
	}
	export class UserDocumentPermissionRule {
	    kindCode: string;
	    action: string;
	    isAllowed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UserDocumentPermissionRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kindCode = source["kindCode"];
	        this.action = source["action"];
	        this.isAllowed = source["isAllowed"];
	    }
	}
	export class UserSystemPermissionRule {
	    permission: string;
	    isAllowed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UserSystemPermissionRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.permission = source["permission"];
	        this.isAllowed = source["isAllowed"];
	    }
	}
	export class UpdateUserDocumentAccessRequest {
	    userId: string;
	    systemPermissions: UserSystemPermissionRule[];
	    permissions: UserDocumentPermissionRule[];
	
	    static createFrom(source: any = {}) {
	        return new UpdateUserDocumentAccessRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.userId = source["userId"];
	        this.systemPermissions = this.convertValues(source["systemPermissions"], UserSystemPermissionRule);
	        this.permissions = this.convertValues(source["permissions"], UserDocumentPermissionRule);
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
	    departmentId: string;
	    isDocumentParticipant: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UpdateUserRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.login = source["login"];
	        this.fullName = source["fullName"];
	        this.isActive = source["isActive"];
	        this.departmentId = source["departmentId"];
	        this.isDocumentParticipant = source["isDocumentParticipant"];
	    }
	}
	export class UserDocumentAccessProfile {
	    systemPermissions: UserSystemPermissionRule[];
	    permissions: UserDocumentPermissionRule[];
	
	    static createFrom(source: any = {}) {
	        return new UserDocumentAccessProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.systemPermissions = this.convertValues(source["systemPermissions"], UserSystemPermissionRule);
	        this.permissions = this.convertValues(source["permissions"], UserDocumentPermissionRule);
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

export namespace services {
	
	export class AdminAuditLogService {
	
	
	    static createFrom(source: any = {}) {
	        return new AdminAuditLogService(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

