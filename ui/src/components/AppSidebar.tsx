import { motion } from "framer-motion"
import {
    LayoutDashboard,
    Settings,
    Activity,
    BookOpen,
    ChevronDown,
    X,
    Shield
} from "lucide-react"

interface AppSidebarProps {
    isOpen: boolean
    onClose: () => void
    onNavigate: (page: string) => void
}

export function AppSidebar({ isOpen, onClose, onNavigate }: AppSidebarProps) {
    const MENU_ITEMS = [
        { icon: LayoutDashboard, label: "Dashboard", id: "dashboard", active: true },
        { icon: Activity, label: "Global Activity", id: "activity" },
        { icon: Settings, label: "Settings", id: "settings" },
        { icon: BookOpen, label: "Documentation", id: "docs" },
    ]

    return (
        <>
            {/* Backdrop */}
            {isOpen && (
                <motion.div
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    exit={{ opacity: 0 }}
                    className="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm"
                    onClick={onClose}
                />
            )}

            {/* Sidebar */}
            <motion.div
                initial={{ x: "-100%" }}
                animate={{ x: isOpen ? 0 : "-100%" }}
                transition={{ type: "spring", damping: 25, stiffness: 200 }}
                className="fixed top-0 left-0 h-full w-64 z-50 bg-[#09090b] border-r border-zinc-800 flex flex-col"
            >
                {/* Header / Project Switcher */}
                <div className="h-14 border-b border-zinc-800 flex items-center px-4 justify-between">
                    <div className="flex items-center gap-2 font-semibold text-white">
                        <div className="w-6 h-6 rounded bg-white flex items-center justify-center">
                            <Shield className="w-4 h-4 text-black fill-current" />
                        </div>
                        <span className="tracking-tight">StackSnap</span>
                    </div>
                    <button onClick={onClose} className="p-1 hover:bg-zinc-800 rounded text-zinc-500 hover:text-white">
                        <X className="w-4 h-4" />
                    </button>
                </div>

                <div className="p-4 flex-1">
                    {/* Project Context */}
                    <div className="mb-6">
                        <div className="text-xs font-medium text-zinc-500 mb-2 uppercase tracking-wider px-2">Workspace</div>
                        <button className="w-full flex items-center justify-between p-2 rounded-md bg-zinc-900 border border-zinc-800 hover:border-zinc-700 transition-colors text-sm text-zinc-200 group">
                            <span className="flex items-center gap-2">
                                <div className="w-2 h-2 rounded-full bg-emerald-500 shadow-[0_0_8px_#10b981]" />
                                Production
                            </span>
                            <ChevronDown className="w-4 h-4 text-zinc-500 group-hover:text-zinc-300" />
                        </button>
                    </div>

                    {/* Navigation */}
                    <nav className="space-y-1">
                        <div className="text-xs font-medium text-zinc-500 mb-2 uppercase tracking-wider px-2">Menu</div>
                        {MENU_ITEMS.map((item) => (
                            <button
                                key={item.label}
                                onClick={() => {
                                    onNavigate(item.id)
                                    onClose()
                                }}
                                className={`w-full flex items-center gap-3 px-2 py-2 rounded-md text-sm font-medium transition-all ${item.active ? 'bg-zinc-800 text-white' : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'}`}
                            >
                                <item.icon className="w-4 h-4" />
                                {item.label}
                            </button>
                        ))}
                    </nav>


                </div>

                {/* Footer Removed as requested */}
            </motion.div>
        </>
    )
}
