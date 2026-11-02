// IExplorerCommand COM class for the Windows 11 top-level "Send to phone" item.
// Compiled by win11/install.ps1 (csc, .NET Framework 4.8).
using System;
using System.Diagnostics;
using System.IO;
using System.Runtime.InteropServices;
using Microsoft.Win32;

[ComVisible(true)]
[Guid(SendToPhoneCommand.Clsid)]
[ClassInterface(ClassInterfaceType.None)]
public sealed class SendToPhoneCommand : IExplorerCommand
{
    public const string Clsid = "E7B8C4A1-5D3E-4F2A-9C1B-8A6D4E2F9012";

    public int GetTitle(IntPtr psiItemArray, out IntPtr ppszName)
    {
        ppszName = Marshal.StringToCoTaskMemUni("Send with Air-Link");
        return 0;
    }

    public int GetIcon(IntPtr psiItemArray, out IntPtr ppszIcon)
    {
        string exe = ExePath();
        if (string.IsNullOrEmpty(exe))
        {
            ppszIcon = IntPtr.Zero;
            return 1; // S_FALSE
        }
        ppszIcon = Marshal.StringToCoTaskMemUni(exe);
        return 0;
    }

    public int GetToolTip(IntPtr psiItemArray, out IntPtr ppszInfotip)
    {
        ppszInfotip = Marshal.StringToCoTaskMemUni("Send the selected files to your phone");
        return 0;
    }

    public int GetCanonicalName(out Guid pguidCommandName)
    {
        pguidCommandName = new Guid(Clsid);
        return 0;
    }

    public int GetState(IntPtr psiItemArray, [MarshalAs(UnmanagedType.Bool)] bool fOkToBeSlow, out uint pCmdState)
    {
        pCmdState = 0; // ECS_ENABLED
        return 0;
    }

    public int Invoke(IntPtr psiItemArray, IntPtr pbc)
    {
        string exe = ExePath();
        if (string.IsNullOrEmpty(exe) || !File.Exists(exe))
            return unchecked((int)0x80004005);

        string args = "--";
        if (psiItemArray != IntPtr.Zero)
        {
            var arr = (IShellItemArray)Marshal.GetTypedObjectForIUnknown(psiItemArray, typeof(IShellItemArray));
            uint n;
            arr.GetCount(out n);
            for (uint i = 0; i < n; i++)
            {
                IShellItem item;
                arr.GetItemAt(i, out item);
                IntPtr ppsz;
                item.GetDisplayName(unchecked((uint)0x80058000), out ppsz); // SIGDN_FILESYSPATH
                string path = Marshal.PtrToStringUni(ppsz);
                Marshal.FreeCoTaskMem(ppsz);
                if (!string.IsNullOrEmpty(path))
                    args += " \"" + path.Replace("\"", "\"\"") + "\"";
            }
        }
        Process.Start(new ProcessStartInfo
        {
            FileName = exe,
            Arguments = args,
            UseShellExecute = false
        });
        return 0;
    }

    public int GetFlags(out uint pFlags)
    {
        pFlags = 0;
        return 0;
    }

    public int EnumSubCommands(out IntPtr ppEnum)
    {
        ppEnum = IntPtr.Zero;
        return 1; // S_FALSE
    }

    static string ExePath()
    {
        try
        {
            using (var k = Registry.CurrentUser.OpenSubKey(@"Software\SendToPhone"))
            {
                object v = k == null ? null : k.GetValue("Exe");
                return v == null ? "" : v.ToString();
            }
        }
        catch { return ""; }
    }
}

[ComImport]
[Guid("a08ce4d0-fa25-44ab-b57c-c7b1c323e0b9")]
[InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
public interface IExplorerCommand
{
    [PreserveSig] int GetTitle(IntPtr psiItemArray, out IntPtr ppszName);
    [PreserveSig] int GetIcon(IntPtr psiItemArray, out IntPtr ppszIcon);
    [PreserveSig] int GetToolTip(IntPtr psiItemArray, out IntPtr ppszInfotip);
    [PreserveSig] int GetCanonicalName(out Guid pguidCommandName);
    [PreserveSig] int GetState(IntPtr psiItemArray, [MarshalAs(UnmanagedType.Bool)] bool fOkToBeSlow, out uint pCmdState);
    [PreserveSig] int Invoke(IntPtr psiItemArray, IntPtr pbc);
    [PreserveSig] int GetFlags(out uint pFlags);
    [PreserveSig] int EnumSubCommands(out IntPtr ppEnum);
}

[ComImport]
[Guid("43826d1e-e718-42ee-bc55-a1e261c37bfe")]
[InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
interface IShellItem
{
    void BindToHandler(IntPtr pbc, [In] ref Guid bhid, [In] ref Guid riid, out IntPtr ppv);
    void GetParent(out IShellItem ppsi);
    void GetDisplayName(uint sigdnName, out IntPtr ppszName);
    void GetAttributes(uint sfgaoMask, out uint psfgaoAttribs);
    void Compare(IShellItem psi, uint hint, out int piOrder);
}

[ComImport]
[Guid("b63ea76d-1f85-456f-a19c-48159efa858b")]
[InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
interface IShellItemArray
{
    void BindToHandler(IntPtr pbc, [In] ref Guid bhid, [In] ref Guid riid, out IntPtr ppvOut);
    void GetPropertyStore(int flags, [In] ref Guid riid, out IntPtr ppv);
    void GetPropertyDescriptionList([In] ref PROPERTYKEY keyType, [In] ref Guid riid, out IntPtr ppv);
    void GetAttributes(int attribFlags, uint sfgaoMask, out uint psfgaoAttribs);
    void GetCount(out uint pdwNumItems);
    void GetItemAt(uint dwIndex, out IShellItem ppsi);
    void EnumItems(out IntPtr ppenumShellItems);
}

[StructLayout(LayoutKind.Sequential, Pack = 4)]
struct PROPERTYKEY
{
    public Guid fmtid;
    public uint pid;
}
