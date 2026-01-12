import { motion } from "framer-motion"
import {
    X,
    Activity,
    Server,
    Database,
    HardDrive,
    CheckCircle,
    AlertTriangle,
    Clock,
    TrendingUp,
    Shield,
    Layers,
    Cpu
} from "lucide-react"

interface GlobalActivityProps {
    isOpen: boolean
    onClose: () => void
    stacks: any[]
    stats: any
    history?: any[]
}

export function GlobalActivity({ isOpen, onClose, stats, history = [] }: GlobalActivityProps) {
    if (!isOpen) return null


    const formatBytes = (bytes: number) => {
        if (!bytes) return "0 B"
        const k = 1024
        const sizes = ["B", "KB", "MB", "GB", "TB"]
        const i = Math.floor(Math.log(bytes) / Math.log(k))
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i]
    }

    const { system } = stats || {}


    const timeline = history.slice(0, 10).map((item: any) => {
        const date = new Date(item.LastModified)
        const timeAgo = Math.round((new Date().getTime() - date.getTime()) / 60000)
        let timeString = timeAgo < 60 ? `${timeAgo} mins ago` : `${Math.floor(timeAgo / 60)} hours ago`
        return {
            id: item.Key,
            type: "backup",
            status: item.verification?.verified ? "success" : (item.verification ? "warning" : "info"),
            stack: item.Key.split('_')[0] || "Unknown",
            time: timeString,
            duration: formatBytes(item.Size)
        }
    })

    return (
        <motion.div
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.95 }}
            className="fixed inset-0 z-50 bg-[#09090b] text-white overflow-y-auto"
        >
            {/* Header */}
            <header className="sticky top-0 z-50 flex items-center justify-between px-8 py-6 bg-[#09090b]/80 backdrop-blur-md border-b border-white/5">
                <div className="flex items-center gap-4">
                    <div className="p-3 rounded-xl bg-indigo-500/10 text-indigo-400">
                        <Activity className="w-6 h-6" />
                    </div>
                    <div>
                        <h1 className="text-2xl font-bold tracking-tight">Mission Control</h1>
                        <p className="text-zinc-400 text-sm">Real-time system telemetry and backup logs</p>
                    </div>
                </div>
                <button
                    onClick={onClose}
                    className="p-2 rounded-full hover:bg-white/10 text-zinc-400 hover:text-white transition-colors"
                >
                    <X className="w-6 h-6" />
                </button>
            </header>

            <div className="max-w-7xl mx-auto p-8 space-y-8">
                {/* Stats Grid */}
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                    <div className="p-6 rounded-2xl bg-zinc-900/50 border border-white/5 space-y-4">
                        <div className="flex items-center justify-between text-zinc-400">
                            <span className="text-sm font-medium">Platform</span>
                            <Server className="w-4 h-4" />
                        </div>
                        <div className="text-2xl font-bold truncate">{system?.os_type || "Docker"} / {system?.architecture || "x64"}</div>
                        <div className="text-xs text-emerald-400 flex items-center gap-1">
                            <CheckCircle className="w-3 h-3" /> Core System Online
                        </div>
                    </div>

                    <div className="p-6 rounded-2xl bg-zinc-900/50 border border-white/5 space-y-4">
                        <div className="flex items-center justify-between text-zinc-400">
                            <span className="text-sm font-medium">Memory Capacity</span>
                            <Cpu className="w-4 h-4" />
                        </div>
                        <div className="text-3xl font-bold text-indigo-500">
                            {formatBytes(system?.memory_total)}
                        </div>
                        <div className="text-xs text-zinc-500">{system?.cpu_cores || 0} CPU Cores Detected</div>
                    </div>

                    <div className="p-6 rounded-2xl bg-zinc-900/50 border border-white/5 space-y-4">
                        <div className="flex items-center justify-between text-zinc-400">
                            <span className="text-sm font-medium">Total Storage Used</span>
                            <HardDrive className="w-4 h-4" />
                        </div>
                        <div className="text-3xl font-bold">{stats?.storage_used || "0 B"}</div>
                        <div className="w-full h-1 bg-zinc-800 rounded-full overflow-hidden">
                            <div className="h-full bg-indigo-500 w-[25%]" />
                        </div>
                    </div>

                    <div className="p-6 rounded-2xl bg-zinc-900/50 border border-white/5 space-y-4">
                        <div className="flex items-center justify-between text-zinc-400">
                            <span className="text-sm font-medium">Total Backups</span>
                            <Database className="w-4 h-4" />
                        </div>
                        <div className="text-4xl font-bold">{stats?.total_backups || 0}</div>
                        <div className="text-xs text-zinc-500">{timeline.length} recent events</div>
                    </div>
                </div>

                {/* Main Content Area */}
                <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
                    {/* Activity Feed */}
                    <div className="lg:col-span-2 space-y-6">
                        <h2 className="text-xl font-semibold flex items-center gap-2">
                            <Clock className="w-5 h-5 text-zinc-500" />
                            Live Activity Feed
                        </h2>

                        <div className="space-y-4">
                            {timeline.length === 0 && (
                                <div className="text-center py-10 text-zinc-500 bg-zinc-900/20 rounded-xl border border-white/5">
                                    No activity recorded yet.
                                </div>
                            )}
                            {timeline.map((item: any) => (
                                <div key={item.id} className="group p-4 rounded-xl bg-zinc-900/30 border border-white/5 hover:border-white/10 transition-all flex items-center justify-between">
                                    <div className="flex items-center gap-4">
                                        <div className={`w-10 h-10 rounded-full flex items-center justify-center ${item.status === 'success' ? 'bg-emerald-500/10 text-emerald-500' :
                                            item.status === 'warning' ? 'bg-amber-500/10 text-amber-500' :
                                                'bg-blue-500/10 text-blue-500'
                                            }`}>
                                            {item.type === 'backup' && <Database className="w-5 h-5" />}
                                            {item.type === 'restore' && <TrendingUp className="w-5 h-5" />}
                                            {item.type === 'scan' && <Shield className="w-5 h-5" />}
                                        </div>
                                        <div>
                                            <div className="font-medium text-white group-hover:text-indigo-400 transition-colors">
                                                Backup: {item.stack}
                                            </div>
                                            <div className="text-xs text-zinc-500 flex items-center gap-2">
                                                <span>{item.time}</span>
                                                <span>•</span>
                                                <span>Size: {item.duration}</span>
                                            </div>
                                        </div>
                                    </div>
                                    <div className={`px-3 py-1 rounded-full text-xs font-medium ${item.status === 'success' ? 'bg-emerald-500/10 text-emerald-500' :
                                        item.status === 'warning' ? 'bg-amber-500/10 text-amber-500' :
                                            'bg-blue-500/10 text-blue-500'
                                        }`}>
                                        {item.status.toUpperCase()}
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>

                    {/* Quick Actions / System Health */}
                    <div className="space-y-6">
                        <h2 className="text-xl font-semibold flex items-center gap-2">
                            <AlertTriangle className="w-5 h-5 text-zinc-500" />
                            Docker Engine Status
                        </h2>

                        <div className="p-6 rounded-2xl bg-gradient-to-br from-zinc-900 to-black border border-white/5 space-y-6">
                            <div className="space-y-4">
                                <div>
                                    <div className="flex justify-between text-sm mb-2">
                                        <span className="text-zinc-400">Running Containers</span>
                                        <span className="text-emerald-400 font-mono">{system?.containers_running || 0} / {system?.containers || 0}</span>
                                    </div>
                                    <div className="h-2 bg-zinc-800 rounded-full overflow-hidden">
                                        <div
                                            className="h-full bg-emerald-500 transition-all duration-500"
                                            style={{ width: `${system?.containers ? (system.containers_running / system.containers) * 100 : 0}%` }}
                                        />
                                    </div>
                                </div>
                                <div>
                                    <div className="flex justify-between text-sm mb-2">
                                        <span className="text-zinc-400">Total Images</span>
                                        <span className="text-blue-400 font-mono">{system?.images_count || 0}</span>
                                    </div>
                                    <div className="h-2 bg-zinc-800 rounded-full overflow-hidden">
                                        <div className="h-full bg-blue-500 w-full opacity-50" />
                                    </div>
                                </div>

                                <div className="pt-4 border-t border-white/5">
                                    <div className="flex justify-between text-sm mb-1">
                                        <span className="text-zinc-500 flex items-center gap-2"><Layers className="w-3 h-3" /> Disk Usage (Docker)</span>
                                    </div>
                                    <div className="text-xl font-mono text-white">
                                        {formatBytes(system?.disk_usage_bytes || 0)}
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </motion.div>
    )
}
