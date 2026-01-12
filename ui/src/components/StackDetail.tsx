import { useState } from "react"
import { motion } from "framer-motion"
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer } from "recharts"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Box, RotateCcw, X, ChevronRight, History } from "lucide-react"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { VerificationReceipt } from "./VerificationReceipt"
import { RestoreModal } from "./RestoreModal"

interface StackDetailProps {
    stack: any
    history: any[]
    onClose: () => void
    onRestore: (filename: string) => void
    onVerify: (filename: string) => void
    onRemove: (path: string) => void
    onBackup: (options?: { pause?: boolean, include_db?: boolean, verify?: boolean, snapshot_images?: boolean }) => void
    isBackingUp: boolean
}

export function StackDetail({ stack, history, onClose, onRestore, onVerify, onRemove, onBackup, isBackingUp }: StackDetailProps) {
    const [activeTab, setActiveTab] = useState("overview")
    const [selectedService, setSelectedService] = useState<string | null>(null)
    const [logs, setLogs] = useState<string>("")
    const [loadingLogs, setLoadingLogs] = useState(false)
    const [pauseContainers, setPauseContainers] = useState(false)
    const [includeDatabases, setIncludeDatabases] = useState(true)
    const [autoVerify, setAutoVerify] = useState(true)
    const [includeAppCode, setIncludeAppCode] = useState(false)
    const [restoreModalOpen, setRestoreModalOpen] = useState(false)
    const [selectedBackup, setSelectedBackup] = useState<any>(null)


    const handleBackupTrigger = (options: any) => {
        onBackup(options)
    }


    const handleRestoreTrigger = (key: string) => {
        setRestoreModalOpen(false)
        onRestore(key)
    }


    const chartData = (history || []).slice().reverse().map(item => ({
        name: new Date(item.LastModified).toLocaleDateString(),
        size: Number((item.Size / 1024 / 1024).toFixed(2)),
        raw: item
    }))

    const fetchLogs = (serviceName: string) => {
        const info = stack.Services[serviceName]
        if (!info?.container_id) return
        setSelectedService(serviceName)
        setLoadingLogs(true)
        setLogs("Loading logs...")
        fetch(`http://localhost:8080/api/logs?id=${info.container_id}`)
            .then(res => res.text())
            .then(data => setLogs(data || "No logs found."))
            .catch(() => setLogs("Failed to fetch logs."))
            .finally(() => setLoadingLogs(false))
    }

    const getHealthColor = (health: string, state: string) => {
        if (state !== "running") return "bg-zinc-500"
        if (health === "healthy") return "bg-emerald-500"
        if (health === "unhealthy") return "bg-red-500"
        if (health === "starting") return "bg-amber-500"
        return "bg-emerald-500"
    }

    return (
        <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 z-50 flex justify-end bg-black/60 backdrop-blur-[2px]"
            onClick={onClose}
        >
            <motion.div
                initial={{ x: "100%" }}
                animate={{ x: 0 }}
                exit={{ x: "100%" }}
                transition={{ type: "spring", damping: 25, stiffness: 200 }}
                className="w-full max-w-2xl h-full bg-zinc-950 border-l border-zinc-800 shadow-2xl flex flex-col"
                onClick={(e) => e.stopPropagation()}
            >
                {/* Header */}
                <div className="h-14 border-b border-zinc-800 flex items-center justify-between px-6 bg-zinc-950/50 backdrop-blur-md sticky top-0 z-10">
                    <div className="flex items-center gap-3">
                        <div className={`w-2 h-2 rounded-full ${stack.Status === "Running" ? "bg-emerald-500 shadow-[0_0_8px_#10b981]" : "bg-red-500"}`} />
                        <span className="font-semibold text-sm tracking-tight">{stack.Name}</span>
                        <span className="text-zinc-600">/</span>
                        <span className="text-xs font-mono text-zinc-500">{stack.ComposeFile.split('/').slice(-2)[0]}</span>
                    </div>
                    <div className="flex items-center gap-2">
                        <Button variant="ghost" size="icon" onClick={onClose} className="h-8 w-8 text-zinc-400 hover:text-white">
                            <X className="w-4 h-4" />
                        </Button>
                    </div>
                </div>

                {/* Tabs */}
                <Tabs value={activeTab} onValueChange={setActiveTab} className="flex-1 flex flex-col min-h-0">
                    <div className="px-6 border-b border-zinc-800">
                        <TabsList className="bg-transparent h-10 gap-6 p-0 text-zinc-400">
                            <TabsTrigger value="overview" className="data-[state=active]:bg-transparent data-[state=active]:text-white data-[state=active]:shadow-none data-[state=active]:border-b-2 data-[state=active]:border-white rounded-none h-10 px-0 font-medium text-xs">
                                Overview
                            </TabsTrigger>
                            <TabsTrigger value="services" className="data-[state=active]:bg-transparent data-[state=active]:text-white data-[state=active]:shadow-none data-[state=active]:border-b-2 data-[state=active]:border-white rounded-none h-10 px-0 font-medium text-xs">
                                Services & Logs
                            </TabsTrigger>
                            <TabsTrigger value="history" className="data-[state=active]:bg-transparent data-[state=active]:text-white data-[state=active]:shadow-none data-[state=active]:border-b-2 data-[state=active]:border-white rounded-none h-10 px-0 font-medium text-xs">
                                Backups
                            </TabsTrigger>
                        </TabsList>
                    </div>

                    <div className="flex-1 overflow-y-auto bg-zinc-950/50">

                        {/* OVERVIEW TAB */}
                        <TabsContent value="overview" className="p-6 m-0 space-y-6">

                            {/* Backup Control Card */}
                            <div className="bg-zinc-900/40 border border-zinc-800 rounded-lg p-5 space-y-4">
                                <div className="flex items-center justify-between">
                                    <h3 className="text-sm font-medium text-zinc-200">Manual Backup</h3>
                                    {isBackingUp && <span className="text-xs text-emerald-500 animate-pulse">Processing...</span>}
                                </div>
                                <div className="grid grid-cols-2 gap-3">
                                    <label className="flex items-center gap-2 text-xs text-zinc-400 cursor-pointer hover:text-zinc-300">
                                        <input type="checkbox" checked={pauseContainers} onChange={e => setPauseContainers(e.target.checked)} className="rounded border-zinc-700 bg-zinc-800 text-white accent-white" />
                                        Stop App (Safe Mode)
                                    </label>
                                    <label className="flex items-center gap-2 text-xs text-zinc-400 cursor-pointer hover:text-zinc-300">
                                        <input type="checkbox" checked={includeDatabases} onChange={e => setIncludeDatabases(e.target.checked)} className="rounded border-zinc-700 bg-zinc-800 text-white accent-white" />
                                        Dump SQL Databases
                                    </label>
                                    <label className="flex items-center gap-2 text-xs text-zinc-400 cursor-pointer hover:text-zinc-300">
                                        <input type="checkbox" checked={autoVerify} onChange={e => setAutoVerify(e.target.checked)} className="rounded border-zinc-700 bg-zinc-800 text-white accent-white" />
                                        Verify Integrity
                                    </label>
                                    <label className="flex items-center gap-2 text-xs text-zinc-400 cursor-pointer hover:text-zinc-300">
                                        <input type="checkbox" checked={includeAppCode} onChange={e => setIncludeAppCode(e.target.checked)} className="rounded border-zinc-700 bg-zinc-800 text-white accent-white" />
                                        Full Snapshot (Slow)
                                    </label>
                                </div>
                                <Button className="w-full bg-white text-black hover:bg-zinc-200" onClick={() => handleBackupTrigger({ pause: pauseContainers, include_db: includeDatabases, verify: autoVerify, snapshot_images: includeAppCode })} disabled={isBackingUp}>
                                    {isBackingUp ? "Backing up..." : "Create Backup Now"}
                                </Button>
                            </div>

                            {/* Stats Grid */}
                            <div className="grid grid-cols-2 gap-4">
                                <div className="p-4 rounded-lg border border-zinc-800 bg-zinc-900/20">
                                    <div className="text-zinc-500 text-xs font-mono uppercase mb-1">Total Size</div>
                                    <div className="text-2xl font-semibold">{chartData.length > 0 ? chartData[0].size + ' MB' : '0 MB'}</div>
                                </div>
                                <div className="p-4 rounded-lg border border-zinc-800 bg-zinc-900/20">
                                    <div className="text-zinc-500 text-xs font-mono uppercase mb-1">Services</div>
                                    <div className="text-2xl font-semibold">{Object.keys(stack.Services || {}).length}</div>
                                </div>
                            </div>

                            {/* Storage Chart */}
                            <div className="h-64 w-full border border-zinc-800 rounded-lg p-4 bg-zinc-900/10">
                                <div className="text-xs text-zinc-500 mb-4">Storage Trend (Last 30 Days)</div>
                                <ResponsiveContainer width="100%" height="100%">
                                    <AreaChart data={chartData}>
                                        <defs>
                                            <linearGradient id="colorSize" x1="0" y1="0" x2="0" y2="1">
                                                <stop offset="5%" stopColor="#fff" stopOpacity={0.1} />
                                                <stop offset="95%" stopColor="#fff" stopOpacity={0} />
                                            </linearGradient>
                                        </defs>
                                        <XAxis dataKey="name" fontSize={10} stroke="#52525b" tickLine={false} axisLine={false} />
                                        <YAxis fontSize={10} stroke="#52525b" tickLine={false} axisLine={false} />
                                        <Tooltip contentStyle={{ backgroundColor: "#09090b", borderColor: "#27272a", borderRadius: "6px", fontSize: "12px" }} itemStyle={{ color: "#fff" }} />
                                        <Area type="monotone" dataKey="size" stroke="#fff" strokeWidth={1} fillOpacity={1} fill="url(#colorSize)" />
                                    </AreaChart>
                                </ResponsiveContainer>
                            </div>

                            {!stack.IsStandalone && (
                                <div className="pt-8 border-t border-zinc-800">
                                    <h4 className="text-xs font-medium text-zinc-500 uppercase tracking-widest mb-4">Danger Zone</h4>
                                    <Button variant="outline" className="w-full border-red-900/30 text-red-700 hover:bg-red-950/10 hover:text-red-600 hover:border-red-900/50 justify-between group" onClick={() => onRemove(stack.ComposeFile.replace("/docker-compose.yml", ""))}>
                                        Stop Monitoring Stack
                                        <ChevronRight className="w-4 h-4 opacity-50 group-hover:opacity-100" />
                                    </Button>
                                </div>
                            )}
                        </TabsContent>

                        {/* SERVICES TAB */}
                        <TabsContent value="services" className="p-0 m-0">
                            <div className="grid grid-cols-12 h-[calc(100vh-120px)]">
                                {/* Service List (Left) */}
                                <div className="col-span-4 border-r border-zinc-800 overflow-y-auto">
                                    {Object.entries(stack.Services || {}).map(([name, info]: [string, any]) => (
                                        <div
                                            key={name}
                                            onClick={() => fetchLogs(name)}
                                            className={`p-4 border-b border-zinc-800/50 cursor-pointer hover:bg-zinc-900/50 transition-colors ${selectedService === name ? 'bg-zinc-900 border-l-2 border-l-white' : 'border-l-2 border-l-transparent'}`}
                                        >
                                            <div className="flex items-center justify-between mb-1">
                                                <span className={`text-sm font-medium ${selectedService === name ? 'text-white' : 'text-zinc-400'}`}>{name}</span>
                                                <div className={`w-1.5 h-1.5 rounded-full ${getHealthColor(info.health, info.state)}`} />
                                            </div>
                                            <div className="flex items-center justify-between text-[10px] text-zinc-600 uppercase">
                                                <span>{info.state}</span>
                                                <span>{info.restart_count > 0 ? `${info.restart_count} restarts` : 'Stable'}</span>
                                            </div>
                                        </div>
                                    ))}
                                </div>

                                {/* Logs (Right) */}
                                <div className="col-span-8 flex flex-col bg-[#0c0c0e]">
                                    <div className="h-10 border-b border-zinc-800 flex items-center justify-between px-4 bg-zinc-950">
                                        <span className="text-xs font-mono text-zinc-500">{selectedService ? `logs: ${selectedService}` : 'Select a service'}</span>
                                        {selectedService && <Button variant="ghost" size="sm" onClick={() => fetchLogs(selectedService)} className="h-6 w-6 p-0 hover:bg-zinc-800"><RotateCcw className="w-3 h-3 text-zinc-500" /></Button>}
                                    </div>
                                    <div className="flex-1 p-4 overflow-auto font-mono text-[11px] leading-relaxed text-zinc-400 whitespace-pre-wrap selection:bg-zinc-700 selection:text-white">
                                        {loadingLogs ? (
                                            <div className="flex items-center gap-2 text-zinc-600">
                                                <div className="w-3 h-3 border-2 border-zinc-600 border-t-white rounded-full animate-spin" />
                                                Fetching logs...
                                            </div>
                                        ) : (
                                            logs || <span className="text-zinc-700">No logs available or container is silent.</span>
                                        )}
                                    </div>
                                </div>
                            </div>
                        </TabsContent>

                        {/* HISTORY TAB */}
                        <TabsContent value="history" className="p-6 m-0">
                            <div className="space-y-4">
                                {(!history || history.length === 0) ? (
                                    <div className="p-8 text-center border border-dashed border-zinc-800 rounded-lg">
                                        <History className="w-8 h-8 text-zinc-700 mx-auto mb-2" />
                                        <p className="text-sm text-zinc-500">No backup history available.</p>
                                    </div>
                                ) : (
                                    history.map((item) => (
                                        <div key={item.Key} className="group relative border border-zinc-800 rounded-lg p-4 bg-zinc-900/20 hover:bg-zinc-900/40 hover:border-zinc-700 transition-all">
                                            <div className="flex items-start justify-between mb-3">
                                                <div className="flex items-center gap-3">
                                                    <div className="p-2 bg-zinc-950 border border-zinc-800 rounded text-zinc-500">
                                                        <Box className="w-4 h-4" />
                                                    </div>
                                                    <div>
                                                        <div className="text-sm font-medium text-zinc-200">Backup Created</div>
                                                        <div className="text-xs text-zinc-500 font-mono">{new Date(item.LastModified).toLocaleString()}</div>
                                                    </div>
                                                </div>
                                                <div className="flex items-center gap-2">
                                                    <span className="text-xs font-mono text-zinc-500">{(item.Size / 1024 / 1024).toFixed(2)} MB</span>
                                                    {/* Status Badge */}
                                                    {item.verification ? (
                                                        item.verification.verified ? (
                                                            <VerificationReceipt verification={item.verification} />
                                                        ) : (
                                                            <Badge variant="destructive" className="h-5 text-[10px] px-1.5" onClick={() => alert(item.verification.error_message)}>
                                                                Failed
                                                            </Badge>
                                                        )
                                                    ) : (
                                                        <Badge variant="outline" className="h-5 text-[10px] px-1.5 text-zinc-500 border-zinc-700">Untested</Badge>
                                                    )}
                                                </div>
                                            </div>

                                            <div className="flex items-center gap-2 pl-[3.25rem]">
                                                <Button size="sm" variant="outline" className="h-7 text-xs border-zinc-700 hover:border-zinc-500 text-zinc-300" onClick={() => {
                                                    setSelectedBackup(item)
                                                    setRestoreModalOpen(true)
                                                }}>
                                                    <RotateCcw className="w-3 h-3 mr-1.5" /> Restore
                                                </Button>
                                                {/* Hidden verify button that appears on hover/untested */}
                                                {!item.verification?.verified && (
                                                    <Button size="sm" variant="ghost" className="h-7 text-xs text-zinc-500 hover:text-white" onClick={() => onVerify(item.Key)}>
                                                        Verify Integrity
                                                    </Button>
                                                )}
                                            </div>
                                        </div>
                                    ))
                                )}
                            </div>
                        </TabsContent>
                    </div>
                </Tabs>
            </motion.div>


            {/* Reuse Restore Modal (It handles its own styling mostly, might need minor tweaks but focused on layout first) */}
            {selectedBackup && (
                <RestoreModal
                    isOpen={restoreModalOpen}
                    onClose={() => {
                        setRestoreModalOpen(false)
                        setSelectedBackup(null)
                    }}
                    onConfirm={() => handleRestoreTrigger(selectedBackup.Key)}
                    stackName={stack.Name}
                    backupKey={selectedBackup.Key}
                    backupDate={new Date(selectedBackup.LastModified).toLocaleString()}
                    estimatedTime={Math.max(1, Math.round(selectedBackup.Size / (10 * 1024 * 1024)))}
                />
            )}
        </motion.div>
    )
}
