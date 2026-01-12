import { useState } from "react"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { AlertTriangle, RotateCcw } from "lucide-react"

interface RestoreModalProps {
    isOpen: boolean
    onClose: () => void
    onConfirm: () => void
    stackName: string
    backupKey: string
    backupDate: string
    estimatedTime: number
}

export function RestoreModal({
    isOpen,
    onClose,
    onConfirm,
    stackName,
    backupKey,
    backupDate,
    estimatedTime
}: RestoreModalProps) {
    const [confirmText, setConfirmText] = useState("")
    const isConfirmed = confirmText === stackName

    const handleConfirm = () => {
        if (isConfirmed) {
            onConfirm()
            setConfirmText("")
            onClose()
        }
    }

    const handleCancel = () => {
        setConfirmText("")
        onClose()
    }

    return (
        <Dialog open={isOpen} onOpenChange={handleCancel}>
            <DialogContent className="max-w-lg">
                <DialogHeader>
                    <DialogTitle className="flex items-center gap-2 text-destructive">
                        <AlertTriangle className="w-5 h-5" />
                        Emergency Restore: {stackName}
                    </DialogTitle>
                    <DialogDescription className="pt-2">
                        You are about to roll back to a previous snapshot. This action cannot be undone.
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4 py-4">
                    {/* Warning Banner */}
                    <div className="p-4 bg-destructive/10 border border-destructive/20 rounded-lg">
                        <div className="flex gap-3">
                            <AlertTriangle className="w-5 h-5 text-destructive shrink-0 mt-0.5" />
                            <div className="space-y-1">
                                <div className="font-semibold text-sm text-destructive">This will overwrite your current data</div>
                                <div className="text-xs text-muted-foreground">
                                    All containers will be stopped, current volumes will be replaced, and the system will be restored to the state captured in this backup.
                                </div>
                            </div>
                        </div>
                    </div>

                    {/* Restore Details */}
                    <div className="space-y-2 p-4 bg-muted/30 rounded-lg border">
                        <div className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Restore Details</div>
                        <div className="space-y-1 font-mono text-sm">
                            <div className="flex justify-between">
                                <span className="text-muted-foreground">Snapshot:</span>
                                <span className="font-semibold">{backupDate}</span>
                            </div>
                            <div className="flex justify-between">
                                <span className="text-muted-foreground">Archive:</span>
                                <span className="text-xs truncate max-w-[200px]" title={backupKey}>{backupKey}</span>
                            </div>
                            <div className="flex justify-between">
                                <span className="text-muted-foreground">Est. Time:</span>
                                <span className="font-semibold">{estimatedTime}s</span>
                            </div>
                        </div>
                    </div>

                    {/* Confirmation Input */}
                    <div className="space-y-2">
                        <Label htmlFor="confirm-input" className="text-sm">
                            Type <span className="font-mono font-bold text-primary">{stackName}</span> to confirm
                        </Label>
                        <Input
                            id="confirm-input"
                            value={confirmText}
                            onChange={(e) => setConfirmText(e.target.value)}
                            placeholder={`Type "${stackName}" here`}
                            className="font-mono"
                            autoComplete="off"
                        />
                    </div>
                </div>

                <DialogFooter className="gap-2">
                    <Button variant="outline" onClick={handleCancel}>
                        Cancel
                    </Button>
                    <Button
                        variant="destructive"
                        onClick={handleConfirm}
                        disabled={!isConfirmed}
                        className="gap-2"
                    >
                        <RotateCcw className="w-4 h-4" />
                        Start Restore Procedure
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    )
}
