import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { Badge } from "@/components/ui/badge"
import { ShieldCheck, CheckCircle2, FileCheck, Database, Box } from "lucide-react"

interface VerificationReceiptProps {
    verification: {
        verified: boolean
        timestamp?: string
        error_message?: string
        container_logs?: string
    }
}

export function VerificationReceipt({ verification }: VerificationReceiptProps) {
    if (!verification.verified) {
        return null
    }

    return (
        <Dialog>
            <DialogTrigger asChild>
                <Badge
                    variant="success"
                    className="text-[9px] h-4 py-0 bg-green-500/10 text-green-500 border-green-500/20 cursor-pointer hover:bg-green-500/20 transition-colors"
                >
                    <ShieldCheck className="w-2.5 h-2.5 mr-1" /> Restorable: Verified
                </Badge>
            </DialogTrigger>
            <DialogContent className="max-w-md">
                <DialogHeader>
                    <DialogTitle className="flex items-center gap-2">
                        <ShieldCheck className="w-5 h-5 text-green-500" />
                        Verification Report
                    </DialogTitle>
                </DialogHeader>

                <div className="space-y-4 py-4">
                    <div className="text-xs text-muted-foreground">
                        {verification.timestamp && (
                            <div className="mb-4 font-mono">
                                Verified: {new Date(verification.timestamp).toLocaleString()}
                            </div>
                        )}
                    </div>

                    <div className="space-y-3">
                        <VerificationCheck
                            icon={<FileCheck className="w-4 h-4" />}
                            label="Archive Integrity"
                            description="Backup file structure validated"
                        />
                        <VerificationCheck
                            icon={<Database className="w-4 h-4" />}
                            label="Database Dumps"
                            description="SQL dumps present and readable"
                        />
                        <VerificationCheck
                            icon={<Box className="w-4 h-4" />}
                            label="Docker Metadata"
                            description="Compose files and configs intact"
                        />
                        <VerificationCheck
                            icon={<CheckCircle2 className="w-4 h-4" />}
                            label="Test Restore"
                            description="Successfully restored to temporary container"
                        />
                    </div>

                    <div className="pt-4 border-t">
                        <div className="flex items-center gap-2 text-sm font-semibold text-green-600">
                            <CheckCircle2 className="w-4 h-4" />
                            This backup is safe to restore
                        </div>
                    </div>
                </div>
            </DialogContent>
        </Dialog>
    )
}

function VerificationCheck({ icon, label, description }: { icon: React.ReactNode, label: string, description: string }) {
    return (
        <div className="flex items-start gap-3 p-2 rounded-md bg-muted/30">
            <div className="text-green-500 mt-0.5">{icon}</div>
            <div className="flex-1">
                <div className="text-sm font-medium">{label}</div>
                <div className="text-xs text-muted-foreground">{description}</div>
            </div>
            <CheckCircle2 className="w-4 h-4 text-green-500 shrink-0 mt-0.5" />
        </div>
    )
}
