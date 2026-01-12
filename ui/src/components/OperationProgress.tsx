import { motion, AnimatePresence } from "framer-motion"
import { Terminal, CheckCircle2, XCircle, Clock, RefreshCw } from "lucide-react"
import { useEffect, useState, useRef } from "react"

interface OperationProgressProps {
    isOpen: boolean
    operation: "backup" | "restore"
    targetName: string
    onComplete?: () => void
    error?: string | null
    logs?: string[]
}

const STEPS = {
    backup: [
        { id: "init", label: "Initializing backup sequence..." },
        { id: "stop", label: "Stopping containers (Safe Mode)..." },
        { id: "dump", label: "Exporting SQL databases..." },
        { id: "volumes", label: "Snapshotting persistent volumes..." },
        { id: "compress", label: "Compressing archive..." },
        { id: "verify", label: "Verifying integrity..." },
        { id: "upload", label: "Finalizing storage..." }
    ],
    restore: [
        { id: "init", label: "Initializing restore sequence..." },
        { id: "download", label: "Downloading archive..." },
        { id: "decrypt", label: "Decrypting parameters..." },
        { id: "stop", label: "Stopping running containers..." },
        { id: "restore_vol", label: "Restoring volumes..." },
        { id: "load_img", label: "Loading container images..." },
        { id: "recreate", label: "Recreating containers..." },
    ]
}

export function OperationProgress({ isOpen, operation, targetName, onComplete, error, logs: externalLogs }: OperationProgressProps) {
    const [currentStep, setCurrentStep] = useState(0)
    const [internalLogs, setInternalLogs] = useState<string[]>([])
    const [progress, setProgress] = useState(0)
    const logsEndRef = useRef<HTMLDivElement>(null)


    const logs = externalLogs || internalLogs


    useEffect(() => {
        if (isOpen) {
            setCurrentStep(0)
            setInternalLogs([])
            setProgress(0)
        }
    }, [isOpen])


    useEffect(() => {
        logsEndRef.current?.scrollIntoView({ behavior: "smooth" })
    }, [logs])


    useEffect(() => {
        if (logs.length > 0) {
            const lastLog = logs[logs.length - 1] || ""

            if (lastLog.includes("Operation Completed Successfully") || lastLog.includes("COMPLETE")) {
                setProgress(100)
                setCurrentStep(STEPS[operation].length - 1)

                if (onComplete) {
                    const timer = setTimeout(onComplete, 1500)
                    return () => clearTimeout(timer)
                }
            }
        }
    }, [logs, onComplete, operation])


    useEffect(() => {
        if (!isOpen || error || externalLogs) return

        const steps = STEPS[operation]
        const interval = setInterval(() => {
            setCurrentStep((prev) => {
                if (prev >= steps.length - 1) {
                    clearInterval(interval)
                    if (onComplete) setTimeout(onComplete, 1000)
                    return prev
                }


                const step = steps[prev + 1]
                setInternalLogs(l => [...l, `[${new Date().toLocaleTimeString()}] INFO: ${step.label}`])

                return prev + 1
            })
        }, 1200)

        const progressInterval = setInterval(() => {
            setProgress(p => {
                if (p >= 100) {
                    clearInterval(progressInterval)
                    return 100
                }
                return p + (100 / (steps.length * 12))
            })
        }, 100)

        setInternalLogs(l => [...l, `[${new Date().toLocaleTimeString()}] INFO: Starting ${operation} operation for ${targetName}`])

        return () => {
            clearInterval(interval)
            clearInterval(progressInterval)
        }
    }, [isOpen, operation, error, targetName, externalLogs])


    useEffect(() => {
        if (externalLogs && isOpen && progress < 100) {

            setProgress(p => Math.min(95, Math.max(p, externalLogs.length * 4)))
        }
    }, [externalLogs, isOpen, progress])

    if (!isOpen) return null

    const steps = STEPS[operation]

    return (
        <AnimatePresence>
            <motion.div
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                className="fixed inset-0 z-[100] bg-zinc-950/80 backdrop-blur-md flex items-center justify-center p-6"
            >
                <div className="w-full max-w-2xl bg-black border border-zinc-800 rounded-xl shadow-2xl overflow-hidden flex flex-col relative">

                    {/* Background sheen effect */}
                    <div className="absolute inset-0 bg-gradient-to-tr from-zinc-900/50 via-transparent to-transparent pointer-events-none" />

                    {/* Header */}
                    <div className="p-6 border-b border-zinc-800 bg-zinc-900/30 flex justify-between items-center relative z-10">
                        <div>
                            <h3 className="text-lg font-semibold text-white flex items-center gap-2">
                                {error ? <XCircle className="text-red-500" /> : <RefreshCw className={`w-5 h-5 ${progress < 100 ? "animate-spin" : ""} text-emerald-500`} />}
                                {error ? "Operation Failed" : (progress >= 100 ? "Operation Complete" : `${operation === 'backup' ? 'Creating Backup' : 'Restoring Stack'}...`)}
                            </h3>
                            <p className="text-xs text-zinc-500 font-mono mt-1">Target: {targetName}</p>
                        </div>
                        <div className="text-right">
                            <div className="text-sm font-mono font-bold text-white">{Math.min(100, Math.round(progress))}%</div>
                            <div className="text-[10px] text-zinc-500 flex items-center justify-end gap-1">
                                <Clock className="w-3 h-3" /> {progress >= 100 ? "Done" : "Processing..."}
                            </div>
                        </div>
                    </div>

                    {/* Progress Bar */}
                    <div className="h-1 bg-zinc-900 w-full">
                        <motion.div
                            className={`h-full ${error ? 'bg-red-500' : 'bg-gradient-to-r from-emerald-600 to-emerald-400'}`}
                            initial={{ width: 0 }}
                            animate={{ width: `${progress}%` }}
                            transition={{ ease: "linear" }}
                        />
                    </div>

                    {/* Content Grid */}
                    <div className="grid grid-cols-1 md:grid-cols-2 h-[300px] relative z-10">

                        {/* Steps List */}
                        <div className="p-6 border-r border-zinc-800 space-y-4 bg-zinc-900/10 overflow-y-auto">
                            {steps.map((step, idx) => (
                                <div key={step.id} className={`flex items-center gap-3 text-sm transition-colors ${idx === currentStep ? 'text-white' : idx < currentStep ? 'text-emerald-500' : 'text-zinc-600'}`}>
                                    {idx < currentStep ? (
                                        <CheckCircle2 className="w-4 h-4" />
                                    ) : idx === currentStep ? (
                                        <div className="w-4 h-4 flex items-center justify-center">
                                            <div className="w-1.5 h-1.5 rounded-full bg-white animate-pulse" />
                                        </div>
                                    ) : (
                                        <div className="w-4 h-4" />
                                    )}
                                    <span className={idx === currentStep ? "font-medium" : ""}>{step.label}</span>
                                </div>
                            ))}
                        </div>

                        {/* Terminal Logs */}
                        <div className="bg-[#0c0c0e] p-4 font-mono text-[10px] text-zinc-400 overflow-y-auto flex flex-col">
                            <div className="flex items-center gap-2 text-zinc-600 mb-2 pb-2 border-b border-zinc-900">
                                <Terminal className="w-3 h-3" />
                                <span>Output Log</span>
                            </div>
                            <div className="space-y-1 flex-1">
                                {logs.map((log, i) => (
                                    <div key={i} className="break-all">
                                        <span className="text-zinc-300">{log}</span>
                                    </div>
                                ))}
                                {error && (
                                    <div className="text-red-400 mt-2">
                                        [ERROR] {error}
                                    </div>
                                )}
                                <div ref={logsEndRef} />
                            </div>
                        </div>

                    </div>

                    {/* Error Actions */}
                    {error && (
                        <div className="p-4 bg-red-950/20 border-t border-red-900/50 flex justify-end">
                            <button onClick={onComplete} className="px-4 py-2 bg-red-600 hover:bg-red-500 text-white rounded-md text-sm font-medium transition-colors">
                                Close
                            </button>
                        </div>
                    )}

                </div>
            </motion.div>
        </AnimatePresence>
    )
}
