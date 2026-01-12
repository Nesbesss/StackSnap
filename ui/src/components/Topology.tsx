import { motion } from "framer-motion"
import { Globe, Server, Database, Cloud } from "lucide-react"

export function Topology() {
    return (
        <div className="w-full flex items-center justify-center p-6 bg-black/50 overflow-x-auto">
            <div className="flex items-center gap-8 relative min-w-max">

                {/* Background Grid - "Schematic" feel */}
                <div className="absolute inset-0 bg-[linear-gradient(to_right,#80808012_1px,transparent_1px),linear-gradient(to_bottom,#80808012_1px,transparent_1px)] bg-[size:14px_14px] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_0%,#000_70%,transparent_100%)] pointer-events-none" />

                {/* Client Node */}
                <div className="relative group z-10">
                    <div className="w-16 h-16 rounded-xl bg-gradient-to-b from-zinc-800 to-zinc-900 border border-zinc-700 flex items-center justify-center shadow-lg group-hover:border-zinc-500 transition-colors">
                        <Globe className="w-6 h-6 text-zinc-400 group-hover:text-white transition-colors" />
                    </div>
                    <div className="absolute -bottom-6 w-full text-center text-[10px] font-mono font-medium text-zinc-500 uppercase tracking-widest">Client</div>
                </div>

                {/* Connection Line 1 */}
                <div className="w-24 h-[1px] bg-zinc-800 relative">
                    <motion.div
                        className="absolute top-1/2 -translate-y-1/2 w-1.5 h-1.5 bg-white rounded-full shadow-[0_0_8px_white]"
                        animate={{ x: [0, 96], opacity: [0, 1, 0] }}
                        transition={{ duration: 2, repeat: Infinity, ease: "linear" }}
                    />
                </div>

                {/* API Node */}
                <div className="relative group z-10">
                    <div className="absolute -inset-px bg-gradient-to-b from-white/20 to-transparent rounded-xl opacity-0 group-hover:opacity-100 transition-opacity" />
                    <div className="w-20 h-20 rounded-xl bg-zinc-900 border border-zinc-700 flex items-center justify-center shadow-2xl relative">
                        <div className="absolute top-2 right-2 flex gap-1">
                            <div className="w-1 h-1 rounded-full bg-emerald-500 shadow-[0_0_4px_#10b981]" />
                        </div>
                        <Server className="w-8 h-8 text-zinc-100" />
                    </div>
                    <div className="absolute -bottom-6 w-full text-center text-[10px] font-mono font-medium text-zinc-500 uppercase tracking-widest">StackSnap</div>
                </div>

                {/* Connection Line 2 */}
                <div className="w-24 h-[1px] bg-zinc-800 relative">
                    <motion.div
                        className="absolute top-1/2 -translate-y-1/2 w-1.5 h-1.5 bg-white rounded-full shadow-[0_0_8px_white]"
                        animate={{ x: [0, 96], opacity: [0, 1, 0] }}
                        transition={{ duration: 2, repeat: Infinity, ease: "linear", delay: 1 }}
                    />
                </div>

                {/* Storage Node (Docker/DB) */}
                <div className="relative group z-10">
                    <div className="w-16 h-16 rounded-xl bg-gradient-to-b from-zinc-800 to-zinc-900 border border-zinc-700 flex items-center justify-center shadow-lg group-hover:border-zinc-500 transition-colors gap-1">
                        <Database className="w-4 h-4 text-zinc-400 group-hover:text-white transition-colors" />
                        <span className="text-zinc-700">/</span>
                        <Cloud className="w-4 h-4 text-zinc-400 group-hover:text-white transition-colors" />
                    </div>
                    <div className="absolute -bottom-6 w-full text-center text-[10px] font-mono font-medium text-zinc-500 uppercase tracking-widest">Storage</div>
                </div>

            </div>
        </div>
    )
}
