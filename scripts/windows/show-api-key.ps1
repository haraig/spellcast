<#
.SYNOPSIS
    Shows the Anthropic API key that Spellcast stored in the Windows Credential Manager.

.DESCRIPTION
    Spellcast stores the key as the generic credential "spellcast:anthropic-api-key",
    with the key as UTF-8 text. The Credential Manager window reads such a value as
    UTF-16 and shows unreadable characters, so use this script to read it.

    The script only reads. It does not change or delete anything. Anyone who can run
    programs as your Windows user can read the key the same way.

.PARAMETER Copy
    Copy the key to the clipboard instead of printing it, so it does not stay in the
    terminal's scrollback. Remember to clear the clipboard afterwards.

.PARAMETER Target
    The credential name. The default is the one Spellcast uses.

.EXAMPLE
    powershell -File scripts\windows\show-api-key.ps1

.EXAMPLE
    powershell -File scripts\windows\show-api-key.ps1 -Copy
#>
[CmdletBinding()]
param(
    [switch]$Copy,
    [string]$Target = "spellcast:anthropic-api-key"
)

Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
using System.Text;

public static class SpellcastCredentialReader
{
    [StructLayout(LayoutKind.Sequential, CharSet = CharSet.Unicode)]
    private struct CREDENTIAL
    {
        public uint Flags;
        public uint Type;
        public string TargetName;
        public string Comment;
        public long LastWritten;
        public uint CredentialBlobSize;
        public IntPtr CredentialBlob;
        public uint Persist;
        public uint AttributeCount;
        public IntPtr Attributes;
        public string TargetAlias;
        public string UserName;
    }

    [DllImport("advapi32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
    private static extern bool CredRead(string target, uint type, uint flags, out IntPtr credential);

    [DllImport("advapi32.dll")]
    private static extern void CredFree(IntPtr credential);

    // Returns null if there is no such credential.
    public static string ReadUtf8(string target)
    {
        IntPtr ptr;
        if (!CredRead(target, 1 /* CRED_TYPE_GENERIC */, 0, out ptr))
        {
            return null;
        }
        try
        {
            var cred = (CREDENTIAL)Marshal.PtrToStructure(ptr, typeof(CREDENTIAL));
            var bytes = new byte[cred.CredentialBlobSize];
            Marshal.Copy(cred.CredentialBlob, bytes, 0, bytes.Length);
            return Encoding.UTF8.GetString(bytes);
        }
        finally
        {
            CredFree(ptr);
        }
    }
}
'@

$key = [SpellcastCredentialReader]::ReadUtf8($Target)
if ([string]::IsNullOrEmpty($key)) {
    [Console]::Error.WriteLine("No stored key found under '$Target'. Set one with the tray entry 'Set API key...'.")
    exit 1
}

if ($Copy) {
    Set-Clipboard -Value $key
    Write-Output "The key was copied to the clipboard."
} else {
    Write-Output $key
}
