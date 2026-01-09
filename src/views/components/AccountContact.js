export default {
    name: 'AccountContact',
    data() {
        return {
            contacts: []
        }
    },
    methods: {
        async openModal() {
            try {
                this.dtClear()
                await this.submitApi();
                $('#modalContactList').modal('show');
                this.dtRebuild()
                showSuccessInfo("Contatos obtidos")
            } catch (err) {
                showErrorInfo(err)
            }
        },
        dtClear() {
            $('#account_contacts_table').DataTable().destroy();
        },
        dtRebuild() {
            $('#account_contacts_table').DataTable({
                "pageLength": 10,
                "reloadData": true,
            }).draw();
        },
        async submitApi() {
            try {
                let response = await window.http.get(`/user/my/contacts`)
                this.contacts = response.data.results.data;
            } catch (error) {
                if (error.response) {
                    throw new Error(error.response.data.message);
                }
                throw new Error(error.message);
            }
        },
        getPhoneNumber(jid) {
            return jid.split('@')[0];
        },
        exportToCSV() {
            if (!this.contacts || this.contacts.length === 0) {
                showErrorInfo("Nenhum contato para exportar");
                return;
            }
            
            // Create CSV content with headers
            let csvContent = "Número de Telefone,Nome\n";
            
            // Add each contact as a row
            this.contacts.forEach(contact => {
                const phoneNumber = this.getPhoneNumber(contact.jid);
                // Escape commas and quotes in the name field
                const escapedName = contact.name ? contact.name.replace(/"/g, '""') : "";
                csvContent += `${phoneNumber},"${escapedName}"\n`;
            });
            
            // Create a Blob with the CSV data
            const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
            
            // Create a download link and trigger download
            const url = URL.createObjectURL(blob);
            const link = document.createElement('a');
            link.setAttribute('href', url);
            link.setAttribute('download', 'contacts.csv');
            link.style.visibility = 'hidden';
            document.body.appendChild(link);
            link.click();
            document.body.removeChild(link);
            
            showSuccessInfo("Contatos exportados para CSV");
        }
    },
    template: `
    <div class="olive card" @click="openModal" style="cursor: pointer">
        <div class="content">
            <a class="ui olive right ribbon label">Contatos</a>
            <div class="header">Meus Contatos</div>
            <div class="description">
                Exibir todos os seus contatos
            </div>
        </div>
    </div>
    
    <!--  Modal Contact List  -->
    <div class="ui large modal" id="modalContactList">
        <i class="close icon"></i>
        <div class="header">
            Meus Contatos
            <button class="ui green right floated button" @click="exportToCSV">
                <i class="download icon"></i> Exportar para CSV
            </button>
        </div>
        <div class="content">
            <table class="ui celled table" id="account_contacts_table">
                <thead>
                <tr>
                    <th>Número de Telefone</th>
                    <th>Nome</th>
                </tr>
                </thead>
                <tbody v-if="contacts != null">
                <tr v-for="contact in contacts">
                    <td>{{ getPhoneNumber(contact.jid) }}</td>
                    <td>{{ contact.name }}</td>
                </tr>
                </tbody>
            </table>
        </div>
    </div>
    `
}
