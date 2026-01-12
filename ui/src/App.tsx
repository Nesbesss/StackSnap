import { useEffect, useState } from "react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Shield, RefreshCw, Settings as SettingsIcon, Play, Box, MoreHorizontal, Menu } from "lucide-react"
import { Topology } from "./components/Topology"
import { StackDetail } from "./components/StackDetail"
import { Settings } from "./components/Settings"
import { Onboarding } from "./pages/Onboarding"
import { AnimatePresence } from "framer-motion"
import { AppSidebar } from "./components/AppSidebar"
import { GlobalActivity } from "./components/GlobalActivity"
import { OperationProgress } from "./components/OperationProgress"

interface Stack {
 Name: string
 Status: string
 Services: Record<string, any>
 ComposeFile: string
 IsStandalone: boolean
}

export default function App() {
 const [stacks, setStacks] = useState<Stack[]>([])
 const [loading, setLoading] = useState(true)
 const [stats, setStats] = useState<any>({ total_backups: 0, success_rate: 100, storage_used: "0 B" })


 const [selectedStack, setSelectedStack] = useState<Stack | null>(null)
 const [stackHistory, setStackHistory] = useState<any[]>([])


 const [_error, setError] = useState("")
 const [backingUp, setBackingUp] = useState<string | null>(null)
 const [showSettings, setShowSettings] = useState(false)
 const [showGlobalActivity, setShowGlobalActivity] = useState(false)
 const [sidebarOpen, setSidebarOpen] = useState(false)


 const [setupRequired, setSetupRequired] = useState(false)
 const [checkingAuth, setCheckingAuth] = useState(true)

 useEffect(() => {
  checkHealth()
 }, [])

 const checkHealth = () => {
  fetch("http://localhost:8080/api/health")
   .then(res => res.json())
   .then(data => {
    if (data.status === "setup_required") {
     setSetupRequired(true)
    } else {
     setSetupRequired(false)
     fetchStacks()
     fetchStats()
    }
    setCheckingAuth(false)
   })
   .catch(err => {
    console.error("Health check failed", err)

    setCheckingAuth(false)
   })
 }


 useEffect(() => {
  if (setupRequired) return
  const interval = setInterval(() => {

   fetchStats()

   if (selectedStack) {
    fetchHistory(selectedStack.Name)
   }
  }, 30000)
  return () => clearInterval(interval)
 }, [selectedStack])

 const fetchStacks = () => {
  fetch("http://localhost:8080/api/stacks")
   .then((res) => {
    if (!res.ok) throw new Error("Failed to fetch stacks")
    return res.json()
   })
   .then((data) => {
    const list = Array.isArray(data) ? data : [data]
    setStacks(list.filter(Boolean))
    setLoading(false)
    setError("")
   })
   .catch((err) => {
    console.error(err)
    setError("Could not connect to StackSnap API. Is the server running?")
    setLoading(false)
   })
 }

 const fetchStats = () => {

  fetch("http://localhost:8080/api/stats")
   .then((res) => res.json())
   .then((data) => setStats((prev: any) => ({ ...prev, ...data })))
   .catch((err) => console.error("Failed to fetch stats", err))


  fetch("http://localhost:8080/api/system-health")
   .then((res) => res.json())
   .then((data) => setStats((prev: any) => ({ ...prev, system: data })))
   .catch((err) => console.error("Failed to fetch system health", err))
 }

 const fetchHistory = (stackName: string) => {
  const url = `http://localhost:8080/api/history?prefix=${encodeURIComponent(stackName)}`
  fetch(url)
   .then((res) => res.json())
   .then((data) => {
    if (Array.isArray(data)) {
     setStackHistory(data)
    } else {
     setStackHistory([])
    }
   })
   .catch(() => setStackHistory([]))
 }

 const [globalHistory, setGlobalHistory] = useState<any[]>([])
 const fetchGlobalHistory = () => {
  fetch("http://localhost:8080/api/history")
   .then((res) => res.json())
   .then((data) => {
    if (Array.isArray(data)) {
     setGlobalHistory(data)
    }
   })
 }

 useEffect(() => {
  if (showGlobalActivity) {
   fetchGlobalHistory()
   fetchStats()
  }
 }, [showGlobalActivity])

 const openStackDetail = (stack: Stack) => {
  setSelectedStack(stack)
  setStackHistory([])
  fetchHistory(stack.Name)
 }

 const handleRestore = (filename: string) => {

  setBackingUp(`Restoring ${filename}...`)
  setProgressActive(true)
  setProgressLogs([])
  setProgressError(null)

  fetch("http://localhost:8080/api/restore", {
   method: "POST",
   body: JSON.stringify({
    filename,
    project_name: selectedStack?.Name
   }),
   headers: { "Content-Type": "application/json" }
  })
   .then((res) => {
    if (!res.ok) throw new Error("Restore failed")

   })
   .catch((err) => {
    setProgressError("Failed to trigger restore: " + err.message)
   })
 }

 const handleAddStack = () => {
  const path = prompt("Enter the absolute path to your Docker Compose directory:")
  if (!path) return

  fetch("http://localhost:8080/api/stacks/add", {
   method: "POST",
   body: JSON.stringify({ path }),
   headers: { "Content-Type": "application/json" }
  })
   .then((res) => {
    if (!res.ok) return res.text().then(text => { throw new Error(text) })
    fetchStacks()
   })
   .catch((err) => alert("Error: " + err.message))
 }

 const handleRemoveStack = (path: string) => {
  if (!confirm("Are you sure you want to stop monitoring this stack? (Containers will keep running)")) return

  fetch("http://localhost:8080/api/stacks/remove", {
   method: "POST",
   body: JSON.stringify({ path }),
   headers: { "Content-Type": "application/json" }
  })
   .then((res) => {
    if (!res.ok) throw new Error("Failed to remove stack")
    fetchStacks()
    setSelectedStack(null)
   })
   .catch((err) => alert("Error: " + err.message))
 }

 const handleVerify = (key: string) => {
  fetch("http://localhost:8080/api/verify", {
   method: "POST",
   body: JSON.stringify({ key }),
   headers: { "Content-Type": "application/json" }
  })
   .then((res) => {
    if (!res.ok) throw new Error("Verification failed to start")
    alert("Verification started! This will take a few seconds.")
    if (selectedStack) fetchHistory(selectedStack.Name)
   })
   .catch((err) => alert("Error: " + err.message))
 }

 const handleBackup = (stackPath: string, projectName: string, opts?: { pause?: boolean, include_db?: boolean, verify?: boolean, snapshot_images?: boolean }) => {
  setBackingUp(stackPath || projectName)

  fetch("http://localhost:8080/api/backups", {
   method: "POST",
   body: JSON.stringify({
    location: stackPath,
    project_name: projectName,
    pause: opts?.pause ?? false,
    include_db: opts?.include_db ?? true,
    verify: opts?.verify ?? false,
    snapshot_images: opts?.snapshot_images ?? false
   }),
   headers: { "Content-Type": "application/json" }
  })
   .then((res) => {
    if (!res.ok) throw new Error("Backup trigger failed")
    return res.json()
   })
   .then(() => {

    setTimeout(() => {
     setBackingUp(null)
     fetchStats()
     if (selectedStack && selectedStack.ComposeFile.includes(stackPath)) {
      fetchHistory(selectedStack.Name)
     }
    }, 2000)
   });
 }


 const [progressLogs, setProgressLogs] = useState<string[]>([])
 const [progressError, setProgressError] = useState<string | null>(null)
 const [progressActive, setProgressActive] = useState(false)

 useEffect(() => {
  if (!setupRequired && !checkingAuth) {
   console.log(" Connecting to Event Stream...")
   const evtSource = new EventSource("http://localhost:8080/api/events")

   evtSource.onmessage = (event) => {
    const msg = event.data
    if (msg === "COMPLETE") {


     fetchStats()
     if (selectedStack) fetchHistory(selectedStack.Name)



     setProgressLogs(prev => [...prev, "[SYSTEM] Operation Completed Successfully"])
     setTimeout(() => {
      setBackingUp(null)
      setProgressActive(false)
     }, 2000)
    } else if (msg.startsWith("ERROR:")) {
     setProgressError(msg.replace("ERROR:", "").trim())
    } else {
     setProgressLogs(prev => [...prev, msg])
    }
   }

   evtSource.onerror = (err) => {
    console.error("EventSource failed:", err)
    evtSource.close()
   }

   return () => {
    evtSource.close()
   }
  }
 }, [setupRequired, checkingAuth, selectedStack])






 if (checkingAuth) {
  return <div className="min-h-screen flex items-center justify-center bg-black text-white">
   <div className="flex flex-col items-center gap-4">
    <Shield className="w-8 h-8 text-white animate-pulse" />
   </div>
  </div>
 }

 if (setupRequired) {
  return <Onboarding onComplete={() => checkHealth()} />
 }

 return (
  <div className="min-h-screen bg-background text-foreground font-sans selection:bg-white/20">
   {/* Settings Modal */}
   <AnimatePresence>
    {showSettings && (
     <Settings
      onClose={() => setShowSettings(false)}
      onSave={() => { fetchStacks(); fetchStats(); }}
     />
    )}
   </AnimatePresence>

   {/* Detail Modal */}
   <AnimatePresence>
    {selectedStack && (
     <StackDetail
      stack={selectedStack}
      history={stackHistory}
      onClose={() => setSelectedStack(null)}
      onRestore={handleRestore}
      onVerify={handleVerify}
      onRemove={handleRemoveStack}
      onBackup={(opts) => handleBackup(
       selectedStack.ComposeFile ? selectedStack.ComposeFile.replace("/docker-compose.yml", "") : "",
       selectedStack.Name,
       opts
      )}
      isBackingUp={backingUp === (selectedStack.ComposeFile ? selectedStack.ComposeFile.replace("/docker-compose.yml", "") : selectedStack.Name)}
     />
    )}
   </AnimatePresence>

   {/* Global Progress Overlay (When backingUp is set) */}
   <AnimatePresence>
    {(backingUp || progressActive) && (
     <OperationProgress
      isOpen={true}
      operation={backingUp ? "backup" : "restore"}
      targetName={backingUp || "Restore Operation"}
      logs={progressLogs}
      error={progressError}
      onComplete={() => {
       setBackingUp(null)
       setProgressActive(false)
       setProgressLogs([])
       setProgressError(null)
      }}
     />
    )}
   </AnimatePresence>

   <AnimatePresence>
    {showGlobalActivity && (
     <GlobalActivity
      isOpen={showGlobalActivity}
      onClose={() => setShowGlobalActivity(false)}
      stacks={stacks}
      stats={stats}
      history={globalHistory}
     />
    )}
   </AnimatePresence>

   <AppSidebar
    isOpen={sidebarOpen}
    onClose={() => setSidebarOpen(false)}
    onNavigate={(page) => {
     if (page === 'settings') setShowSettings(true)
     if (page === 'activity') setShowGlobalActivity(true)
     if (page === 'docs') window.open('https://github.com/nesbes/stacksnap', '_blank')
    }}
   />

   {/* Command Center Header */}
   <header className="fixed top-0 left-0 right-0 h-14 bg-black/50 backdrop-blur-xl border-b border-white/5 z-40 flex items-center justify-between px-6 transition-all duration-300">
    <div className="flex items-center gap-4">
     <button
      onClick={() => setSidebarOpen(true)}
      className="p-2 -ml-2 rounded-md hover:bg-white/10 text-zinc-400 hover:text-white transition-colors"
     >
      <Menu className="w-5 h-5" />
     </button>
     <div className="flex items-center gap-3 text-sm">
      <div className="flex items-center gap-2 font-semibold text-white">
       <Shield className="w-5 h-5" />
       <span className="tracking-tight">StackSnap</span>
      </div>
      <span className="text-zinc-600">/</span>
      <span className="text-zinc-400 hover:text-white cursor-pointer transition-colors">Projects</span>
      <span className="text-zinc-600">/</span>
      <span className="text-white font-medium">Production</span>
     </div>
    </div>
    <div className="ml-auto flex items-center gap-3">
     <div className="text-xs text-muted-foreground mr-2 font-mono hidden sm:block">US-EAST-1</div>
     <Button variant="ghost" size="icon" onClick={() => { fetchStacks(); fetchStats(); }} className="h-8 w-8 text-muted-foreground hover:text-foreground"><RefreshCw className="h-4 w-4" /></Button>
     <Button variant="ghost" size="icon" onClick={() => setShowSettings(true)} className="h-8 w-8 text-muted-foreground hover:text-foreground"><SettingsIcon className="h-4 w-4" /></Button>
     <div className="h-7 w-7 rounded-full bg-white/10 flex items-center justify-center text-[10px] font-medium ring-1 ring-white/20">US</div>
    </div>
   </header>

   {/* Topology / Schematic Header */}
   <div className="pt-14 border-b border-white/5 bg-black/20" >
    <Topology />
   </div >

   <main className="container max-w-[1400px] mx-auto p-6 space-y-8">

    {/* Actions Bar */}
    <div className="flex items-center justify-between">
     <h2 className="text-lg font-medium tracking-tight">Active Stacks</h2>
     <Button size="sm" onClick={handleAddStack} className="h-8 bg-white text-black hover:bg-white/90">
      <RefreshCw className="mr-2 h-3.5 w-3.5" /> Import Stack
     </Button>
    </div>

    {/* Data Grid */}
    <div className="rounded-lg border border-white/10 bg-black/40 overflow-hidden overflow-x-auto">
     {/* Header Row */}
     <div className="min-w-[800px] grid grid-cols-12 gap-4 px-6 py-3 border-b border-white/5 bg-white/5 text-xs font-medium text-muted-foreground uppercase tracking-wider">
      <div className="col-span-4">Project Name</div>
      <div className="col-span-2">Status</div>
      <div className="col-span-2">Services</div>
      <div className="col-span-2">Security</div>
      <div className="col-span-2 text-right">Actions</div>
     </div>

     {/* Loading / Empty States */}
     {loading && <div className="p-8 text-center text-sm text-muted-foreground">Loading topology...</div>}
     {!loading && stacks.length === 0 && <div className="p-8 text-center text-sm text-muted-foreground">No stacks found. Import one to get started.</div>}

     {/* Rows */}
     <div className="divide-y divide-white/5 min-w-[800px]">
      {stacks.map((stack) => (
       <div
        key={stack.Name}
        className="grid grid-cols-12 gap-4 px-6 py-4 items-center hover:bg-white/[0.02] transition-colors cursor-pointer group"
        onClick={() => openStackDetail(stack)}
       >
        {/* Name */}
        <div className="col-span-4 font-medium text-sm flex items-center gap-3">
         <div className={`w-2 h-2 rounded-full ${stack.Status === 'Running' ? 'bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.3)]' : 'bg-red-500'}`} />
         <span className="text-white group-hover:text-white/90">{stack.Name}</span>
        </div>

        {/* Status Text */}
        <div className="col-span-2 text-xs text-muted-foreground font-mono">
         {stack.Status}
        </div>

        {/* Services */}
        <div className="col-span-2 flex items-center gap-2">
         <Box className="w-3.5 h-3.5 text-muted-foreground" />
         <span className="text-sm">{Object.keys(stack.Services || {}).length}</span>
        </div>

        {/* Security/Volumes */}
        <div className="col-span-2">
         {stack.IsStandalone ? (
          <Badge variant="outline" className="text-[10px] border-white/10 text-muted-foreground">Standalone</Badge>
         ) : (
          <Badge variant="secondary" className="text-[10px] bg-emerald-500/10 text-emerald-500 hover:bg-emerald-500/20 border-0">Protected</Badge>
         )}
        </div>

        {/* Actions */}
        <div className="col-span-2 flex justify-end gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
         <Button size="icon" variant="ghost" className="h-7 w-7">
          <Play className="w-3.5 h-3.5" />
         </Button>
         <Button size="icon" variant="ghost" className="h-7 w-7">
          <MoreHorizontal className="w-3.5 h-3.5" />
         </Button>
        </div>
       </div>
      ))}
     </div>
    </div>

    {/* Dense Stats Footer */}
    <div className="grid grid-cols-4 gap-4 pt-4 border-t border-white/5">
     <div className="space-y-1">
      <div className="text-xs text-muted-foreground">Total Backups</div>
      <div className="text-xl font-mono font-medium">{stats.total_backups}</div>
     </div>
     <div className="space-y-1">
      <div className="text-xs text-muted-foreground">Success Rate</div>
      <div className="text-xl font-mono font-medium text-emerald-500">{stats.success_rate}%</div>
     </div>
     <div className="space-y-1">
      <div className="text-xs text-muted-foreground">Storage Used</div>
      <div className="text-xl font-mono font-medium">{stats.storage_used}</div>
     </div>
    </div>

   </main>
  </div >
 )
}


