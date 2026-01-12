import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle, CardDescription, CardFooter } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

import { Shield, HardDrive, Cloud, Server } from "lucide-react"

interface OnboardingProps {
    onComplete: () => void
}

export function Onboarding({ onComplete }: OnboardingProps) {
    const [step, setStep] = useState(1)
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState("")


    const [licenseKey, setLicenseKey] = useState("")
    const [licenseServerURL, setLicenseServerURL] = useState("http://localhost:8081")
    const [showAdvanced, setShowAdvanced] = useState(false)
    const [storageType, setStorageType] = useState("local")


    const [s3Bucket, setS3Bucket] = useState("")
    const [s3Region, setS3Region] = useState("us-east-1")
    const [s3Endpoint, setS3Endpoint] = useState("")


    const [s3AccessKey] = useState("")
    const [s3SecretKey] = useState("")

    const handleFinish = () => {
        setLoading(true)
        setError("")

        const payload = {
            license_key: licenseKey,
            license_server_url: licenseServerURL,
            storage: {
                type: storageType,
                s3_bucket: s3Bucket,
                s3_region: s3Region,
                s3_endpoint: s3Endpoint,
                s3_access_key: s3AccessKey,
                s3_secret_key: s3SecretKey
            }
        }

        fetch("http://localhost:8080/api/setup", {
            method: "POST",
            body: JSON.stringify(payload),
            headers: { "Content-Type": "application/json" }
        })
            .then(async res => {
                let data;
                const contentType = res.headers.get("content-type");
                if (contentType && contentType.includes("application/json")) {
                    data = await res.json();
                } else {
                    data = { message: await res.text() };
                }

                if (!res.ok) throw new Error(data.message || "Setup failed")
                return data
            })
            .then(() => {

                onComplete()
            })
            .catch(err => {
                setError(err.message || "Something went wrong")
                setLoading(false)
            })
    }

    return (
        <div className="min-h-screen bg-background flex items-center justify-center p-4">
            <Card className="w-full max-w-lg shadow-2xl border-primary/20">
                <CardHeader>
                    <div className="flex items-center gap-2 mb-2 justify-center">
                        <Shield className="w-8 h-8 text-primary" />
                        <span className="text-2xl font-bold">StackSnap</span>
                    </div>
                    <CardTitle className="text-center">Welcome Setup</CardTitle>
                    <CardDescription className="text-center">
                        Configure your backup appliance in just a few steps.
                    </CardDescription>
                </CardHeader>
                <CardContent className="space-y-6">
                    {error && (
                        <div className="p-3 bg-destructive/10 text-destructive text-sm rounded-md border border-destructive/20 text-center">
                            {error}
                        </div>
                    )}

                    {step === 1 && (
                        <div className="space-y-4 animate-in fade-in slide-in-from-right-4">
                            <div className="space-y-2">
                                <Label>License Key</Label>
                                <Input
                                    placeholder="PRO-XXXX-XXXX-XXXX"
                                    value={licenseKey}
                                    onChange={e => setLicenseKey(e.target.value)}
                                    className="text-center font-mono tracking-widest"
                                />
                                <p className="text-xs text-muted-foreground text-center">
                                    Enter the key provided in your purchase email.
                                </p>
                            </div>

                            {/* Advanced Section */}
                            <div className="pt-2">
                                <Button
                                    variant="link"
                                    size="sm"
                                    className="text-[10px] text-muted-foreground h-auto p-0"
                                    onClick={() => setShowAdvanced(!showAdvanced)}
                                >
                                    {showAdvanced ? "Hide" : "Advanced: Change License Server"}
                                </Button>

                                {showAdvanced && (
                                    <div className="mt-2 p-3 border rounded-md bg-muted/20 space-y-2 animate-in fade-in zoom-in-95">
                                        <Label className="text-[10px] uppercase tracking-wider text-muted-foreground">License Server URL</Label>
                                        <Input
                                            placeholder="http://localhost:8081"
                                            value={licenseServerURL}
                                            onChange={e => setLicenseServerURL(e.target.value)}
                                            className="h-8 text-xs font-mono"
                                        />
                                        <p className="text-[10px] text-muted-foreground">
                                            Only change this if you are using a custom license endpoint or connecting via a specific IP.
                                        </p>
                                    </div>
                                )}
                            </div>

                            <Button
                                className="w-full mt-4"
                                onClick={() => setStep(2)}
                                disabled={licenseKey.length < 5}
                            >
                                Verify & Continue
                            </Button>
                        </div>
                    )}

                    {step === 2 && (
                        <div className="space-y-4 animate-in fade-in slide-in-from-right-4">
                            <Label>Where should backups go?</Label>

                            <div className="grid grid-cols-2 gap-4">
                                <div
                                    className={`p-4 border rounded-lg cursor-pointer transition-all flex flex-col items-center gap-2 ${storageType === 'local' ? 'border-primary bg-primary/5 ring-1 ring-primary' : 'hover:bg-muted/50'}`}
                                    onClick={() => setStorageType("local")}
                                >
                                    <HardDrive className="w-6 h-6" />
                                    <span className="font-semibold text-sm">Local Disk</span>
                                </div>
                                <div
                                    className={`p-4 border rounded-lg cursor-pointer transition-all flex flex-col items-center gap-2 ${storageType === 's3' ? 'border-primary bg-primary/5 ring-1 ring-primary' : 'hover:bg-muted/50'}`}
                                    onClick={() => setStorageType("s3")}
                                >
                                    <Cloud className="w-6 h-6" />
                                    <span className="font-semibold text-sm">S3 Compatible</span>
                                </div>
                            </div>

                            {storageType === 's3' && (
                                <div className="space-y-3 pt-2 text-sm border-t mt-4">
                                    <div>
                                        <Label className="text-xs">Endpoint URL</Label>
                                        <Input placeholder="http://localhost:4566" value={s3Endpoint} onChange={e => setS3Endpoint(e.target.value)} />
                                    </div>
                                    <div className="grid grid-cols-2 gap-2">
                                        <div>
                                            <Label className="text-xs">Bucket Name</Label>
                                            <Input placeholder="my-backups" value={s3Bucket} onChange={e => setS3Bucket(e.target.value)} />
                                        </div>
                                        <div>
                                            <Label className="text-xs">Region</Label>
                                            <Input placeholder="us-east-1" value={s3Region} onChange={e => setS3Region(e.target.value)} />
                                        </div>
                                    </div>
                                </div>
                            )}

                            <div className="flex gap-2 pt-2">
                                <Button variant="outline" onClick={() => setStep(1)}>Back</Button>
                                <Button className="flex-1" onClick={handleFinish} disabled={loading}>
                                    {loading ? "Configuring..." : "Finish Setup"}
                                </Button>
                            </div>
                        </div>
                    )}

                </CardContent>
                <CardFooter className="flex flex-col gap-2 border-t py-4 bg-muted/20">
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                        <Server className="w-3 h-3" />
                        v1.0.0-BETA
                    </div>
                    <p className="text-[10px] text-muted-foreground text-center">
                        <span className="font-bold text-yellow-600/80"> BETA DISCLAIMER:</span> Using this software involves risk.
                        No warranties are provided. Always manual-test your recovery plan.
                    </p>
                </CardFooter>
            </Card>
        </div>
    )
}
