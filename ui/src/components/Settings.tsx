import { useState, useEffect } from "react"
import { motion } from "framer-motion"
import { Button } from "./ui/button"
import { Input } from "./ui/input"
import { Label } from "./ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "./ui/select"
import { X, Shield, Save, Cloud, HardDrive, AlertCircle } from "lucide-react"

interface SettingsProps {
    onClose: () => void
    onSave: () => void
}

export function Settings({ onClose, onSave }: SettingsProps) {
    const [config, setConfig] = useState<any>(null)
    const [loading, setLoading] = useState(true)
    const [saving, setSaving] = useState(false)
    const [error, setError] = useState("")
    const [testStatus, setTestStatus] = useState<{ type: 'success' | 'error', msg: string } | null>(null)
    const [testing, setTesting] = useState(false)

    useEffect(() => {
        fetch("http://localhost:8080/api/config")
            .then(res => res.json())
            .then(data => {
                setConfig(data)
                setLoading(false)
            })
            .catch(() => {
                setError("Failed to load configuration")
                setLoading(false)
            })
    }, [])

    const testConnection = () => {
        setTesting(true)
        setTestStatus(null)




        fetch("http://localhost:8080/api/config", {
            method: "POST",
            body: JSON.stringify(config),
            headers: { "Content-Type": "application/json" }
        })
            .then(res => {
                if (!res.ok) throw new Error("Failed to apply settings for test")
                return fetch("http://localhost:8080/api/test-storage")
            })
            .then(res => {
                if (!res.ok) return res.text().then(t => { throw new Error(t) })
                return res.json()
            })
            .then(data => setTestStatus({ type: 'success', msg: data.message }))
            .catch(err => setTestStatus({ type: 'error', msg: err.message }))
            .finally(() => setTesting(false))
    }

    const handleSave = () => {
        setSaving(true)
        setError("")

        fetch("http://localhost:8080/api/config", {
            method: "POST",
            body: JSON.stringify(config),
            headers: { "Content-Type": "application/json" }
        })
            .then(res => {
                if (!res.ok) throw new Error("Failed to save settings")
                return res.json()
            })
            .then(() => {
                onSave()
                onClose()
            })
            .catch(err => setError(err.message))
            .finally(() => setSaving(false))
    }

    if (loading) return null

    return (
        <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 z-[60] flex items-center justify-center bg-background/80 backdrop-blur-md p-4"
        >
            <motion.div
                initial={{ scale: 0.95, opacity: 0 }}
                animate={{ scale: 1, opacity: 1 }}
                className="w-full max-w-xl bg-card border rounded-2xl shadow-2xl overflow-hidden"
            >
                <div className="p-6 border-b flex justify-between items-center bg-muted/20">
                    <div className="flex items-center gap-2">
                        <Shield className="w-5 h-5 text-primary" />
                        <h2 className="text-xl font-bold font-heading">Global Settings</h2>
                    </div>
                    <Button variant="ghost" size="icon" onClick={onClose} className="rounded-full">
                        <X className="w-5 h-5" />
                    </Button>
                </div>

                <div className="p-8 space-y-8 max-h-[70vh] overflow-y-auto">
                    {/* Storage Section */}
                    <div className="space-y-6">
                        <div className="flex items-center gap-2 text-sm font-semibold uppercase tracking-wider text-muted-foreground">
                            <Cloud className="w-4 h-4" /> Storage Provider
                        </div>

                        <div className="space-y-4">
                            <div className="space-y-2">
                                <Label>Storage Type</Label>
                                <Select
                                    value={config.storage.type}
                                    onValueChange={(v) => setConfig({ ...config, storage: { ...config.storage, type: v } })}
                                >
                                    <SelectTrigger>
                                        <SelectValue placeholder="Select storage" />
                                    </SelectTrigger>
                                    <SelectContent>
                                        <SelectItem value="local">
                                            <div className="flex items-center gap-2">
                                                <HardDrive className="w-4 h-4" /> Local Storage
                                            </div>
                                        </SelectItem>
                                        <SelectItem value="s3">
                                            <div className="flex items-center gap-2">
                                                <Cloud className="w-4 h-4" /> AWS S3 / LocalStack
                                            </div>
                                        </SelectItem>
                                    </SelectContent>
                                </Select>
                            </div>

                            {config.storage.type === 's3' && (
                                <motion.div
                                    initial={{ opacity: 0, y: -10 }}
                                    animate={{ opacity: 1, y: 0 }}
                                    className="space-y-4 pt-4 border-t"
                                >
                                    <div className="grid grid-cols-2 gap-4">
                                        <div className="space-y-2">
                                            <Label>Bucket Name</Label>
                                            <Input
                                                value={config.storage.s3_bucket}
                                                onChange={(e) => setConfig({ ...config, storage: { ...config.storage, s3_bucket: e.target.value } })}
                                                placeholder="stacksnap-backups"
                                            />
                                        </div>
                                        <div className="space-y-2">
                                            <Label>Region</Label>
                                            <Input
                                                value={config.storage.s3_region}
                                                onChange={(e) => setConfig({ ...config, storage: { ...config.storage, s3_region: e.target.value } })}
                                                placeholder="us-east-1"
                                            />
                                        </div>
                                    </div>
                                    <div className="space-y-2">
                                        <Label>Endpoint URL (Optional for LocalStack)</Label>
                                        <Input
                                            value={config.storage.s3_endpoint}
                                            onChange={(e) => setConfig({ ...config, storage: { ...config.storage, s3_endpoint: e.target.value } })}
                                            placeholder="http://localhost:4566"
                                        />
                                    </div>
                                    <div className="grid grid-cols-2 gap-4">
                                        <div className="space-y-2">
                                            <Label>Access Key</Label>
                                            <Input
                                                value={config.storage.s3_access_key}
                                                onChange={(e) => setConfig({ ...config, storage: { ...config.storage, s3_access_key: e.target.value } })}
                                            />
                                        </div>
                                        <div className="space-y-2">
                                            <Label>Secret Key</Label>
                                            <Input
                                                type="password"
                                                value={config.storage.s3_secret_key}
                                                onChange={(e) => setConfig({ ...config, storage: { ...config.storage, s3_secret_key: e.target.value } })}
                                            />
                                        </div>
                                    </div>
                                    <div className="pt-4 flex items-center justify-between">
                                        <Button
                                            variant="outline"
                                            size="sm"
                                            onClick={testConnection}
                                            disabled={testing}
                                        >
                                            {testing ? "Testing..." : "Test Connection"}
                                        </Button>

                                        {testStatus && (
                                            <div className={`text-[10px] font-bold uppercase ${testStatus.type === 'success' ? 'text-green-500' : 'text-red-500'}`}>
                                                {testStatus.msg}
                                            </div>
                                        )}
                                    </div>
                                </motion.div>
                            )}
                        </div>
                    </div>

                    {/* Advanced Section */}
                    <div className="space-y-6 pt-6 border-t">
                        <div className="flex items-center gap-2 text-sm font-semibold uppercase tracking-wider text-muted-foreground">
                            <Shield className="w-4 h-4" /> Licensing & Updates
                        </div>
                        <div className="space-y-2">
                            <Label>License Server URL</Label>
                            <Input
                                value={config.license_server_url}
                                onChange={(e) => setConfig({ ...config, license_server_url: e.target.value })}
                            />
                        </div>
                    </div>

                    {error && (
                        <div className="p-3 bg-destructive/10 border border-destructive/20 rounded-lg flex gap-2 text-destructive text-sm items-center">
                            <AlertCircle className="w-4 h-4" />
                            {error}
                        </div>
                    )}
                </div>

                <div className="p-6 border-t bg-muted/10 flex justify-end gap-3">
                    <Button variant="ghost" onClick={onClose} disabled={saving}>Cancel</Button>
                    <Button onClick={handleSave} disabled={saving} className="min-w-[120px]">
                        {saving ? "Applying..." : <><Save className="w-4 h-4 mr-2" /> Save Changes</>}
                    </Button>
                </div>
            </motion.div>
        </motion.div>
    )
}
